//go:build github_impl
// +build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	u "actionforge/graph-runner/utils"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

var (
	// Defined by GitHub, all actions can expect the repository being mounted at /github/workspace
	// https://docs.github.com/en/actions/creating-actions/creating-a-docker-container-action#accessing-files-created-by-a-container-action
	defaultWorkspaceMount = "/github/workspace"

	//go:embed gh-action@v1.yml
	ghActionNodeDefinition string
)

type ActionType int

const (
	Docker ActionType = iota
	Node
)

type GhActionNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	actionName      string
	actionType      ActionType // docker or node
	actionRuns      ActionRuns
	actionRunJsPath string
}

func (n *GhActionNode) ExecuteImpl(c core.ExecutionContext) error {
	workspace := os.Getenv("GITHUB_WORKSPACE")
	if workspace == "" {
		return fmt.Errorf("GITHUB_WORKSPACE not set")
	}

	fmt.Printf("✅ Executing GitHub action (%s)\n", n.actionName)

	environ := make([]string, len(os.Environ()))
	for _, env := range os.Environ() {
		// remove all INPUT_ env as they are resolved in the next code block below
		if !strings.HasPrefix(env, "INPUT_") {
			environ = append(environ, env)
		}
	}

	withInputs := ""

	for inputName := range n.Inputs.GetInputDefs() {
		v, err := core.InputValueById[string](c, n.Inputs, inputName)
		if err != nil {
			return err
		}
		v = ReplaceContextVariables(v, n.Inputs.GetInputValues())
		environ = append(environ, fmt.Sprintf("INPUT_%v=%v", strings.ToUpper(string(inputName)), v))

		withInputs += fmt.Sprintf(" %s: %s\n", inputName, v)
	}

	u.LoggerBase.Printf("%sRun '%s (%s)'\n%s%s\n",
		u.LogGhStartGroup,
		n.GetId(),
		n.GetNodeType(),
		withInputs,
		u.LogGhEndGroup,
	)

	// Fetch environment variables from the inputs and add them to the command
	e, err := core.InputValueById[[]string](c, n.Inputs, "env")
	if err == nil {
		environ = append(environ, e...)
	}

	if n.actionType == Docker {
		err = n.ExecuteDocker(c, workspace, environ)
	} else if n.actionType == Node {
		err = n.ExecuteNode(c, workspace, environ)
	} else {
		return fmt.Errorf("unsupported action type: %v", n.actionType)
	}
	if err != nil {
		execErr := n.Executions[ni.Gh_action_v1_Output_exec_err]

		// If the error output is not connected, we can safely fail here
		if execErr == nil {
			return utils.Throw(err)
		}

		err = n.Execute(execErr, c)
		if err != nil {
			return err
		}
	}

	// Transfer the output values from the github action to the node output values
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput != "" {
		b, err := os.ReadFile(githubOutput)
		if err != nil {
			return u.Throw(err)
		}

		outputs, err := parseOutputFile(string(b))
		if err != nil {
			return u.Throw(err)
		}
		for key, value := range outputs {
			err = n.SetOutputValue(c, core.OutputId(key), value)
			if err != nil {
				return u.Throw(err)
			}
		}

		// empty output file for the next run
		err = os.WriteFile(githubOutput, []byte(""), 0644)
		if err != nil {
			return u.Throw(err)
		}
	}

	githubPath := os.Getenv("GITHUB_PATH")
	if githubPath != "" {
		p, err := os.ReadFile(githubPath)
		if err != nil {
			return u.Throw(err)
		}

		path := ""

		lines := strings.Split(string(p), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			path += line + string(os.PathListSeparator)
		}

		path += os.Getenv("PATH")
		err = os.Setenv("PATH", path)
		if err != nil {
			return u.Throw(err)
		}
	}

	err = n.Execute(n.Executions[ni.Gh_action_v1_Output_exec], c)
	if err != nil {
		return err
	}

	return nil
}

func (n *GhActionNode) ExecuteNode(c core.ExecutionContext, workspace string, environ []string) error {

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var nodeBin string
	// TODO: (Seb) This is a hack to find the node binary as it relies on the latest
	// runner version, that might not even be the correct runner. Find a way on how
	// to extract the build constant from the runner.
	runnerVersion, err := findHighestVersionDir(filepath.Join(home, "runners"))
	if err != nil {
		nodeBin = "node" // use default node-version
	} else {
		nodeBin = filepath.Join(home, "runners", runnerVersion, "externals", n.actionRuns.Using, "bin", "node")
	}

	fmt.Printf("Use node binary: %s\n", nodeBin)

	cmd := exec.Command(nodeBin, n.actionRunJsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil
	cmd.Env = environ
	cmd.Dir = workspace
	err = cmd.Run()
	if err != nil {
		return u.Throw(err)
	}
	return nil
}

func (n *GhActionNode) ExecuteDocker(c core.ExecutionContext, workspace string, environ []string) error {

	ContainerDisplayName := fmt.Sprintf("%s_%s", sanitize(n.actionRuns.Image, false), uuid.Must(uuid.NewRandom()).String()[:6])

	ContainerImage := strings.TrimPrefix(n.actionRuns.Image, "docker://")

	ContainerEnvironmentVariables := make(map[string]string)
	for _, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			ContainerEnvironmentVariables[parts[0]] = parts[1]
		}
	}

	ContainerEntryArgs := make([]string, 0)
	for _, arg := range n.actionRuns.Args {
		ContainerEntryArgs = append(ContainerEntryArgs, ReplaceContextVariables(arg, n.Inputs.GetInputValues()))
	}

	ci := ContainerInfo{
		ContainerImage:                ContainerImage,
		ContainerDisplayName:          ContainerDisplayName,
		ContainerWorkDirectory:        defaultWorkspaceMount,
		ContainerEntryPointArgs:       strings.Join(ContainerEntryArgs, " "),
		ContainerEnvironmentVariables: ContainerEnvironmentVariables,
		MountVolumes: []Volume{
			{
				SourceVolumePath: workspace,
				TargetVolumePath: defaultWorkspaceMount,
				ReadOnly:         false,
			},
		},
	}
	exitCode, err := DockerRun(context.Background(), ci, nil, nil)
	if err != nil {
		return u.Throw(err)
	}
	if exitCode != 0 {
		return u.Throw(fmt.Errorf("docker run failed with exit code %d", exitCode))
	}
	return nil
}

func parseNodeTypeUri(nodeTypeUri string) (registry string, owner string, regname string, ref string, err error) {
	if strings.HasPrefix(nodeTypeUri, "http://") || strings.HasPrefix(nodeTypeUri, "https://") {
		// only the frontend deals with `https//:` uris, the backend
		// or gh action must only contain the normalized uri format.
		return "", "", "", "", fmt.Errorf("url must only contain the node path uri, not the full url")
	}

	matches := getNodeTypeUriRegex().FindStringSubmatch(nodeTypeUri)
	if len(matches) == 0 {
		return "", "", "", "", fmt.Errorf("invalid node type id")
	}
	return strings.TrimSuffix(matches[1], "/"), matches[2], matches[3], strings.TrimPrefix(matches[4], "@"), nil
}

func init() {
	// Use the input token that is provided by the user, or passed as default,
	// otherwise fall back to use the runtime token that is provided by github.
	if os.Getenv("INPUT_TOKEN") != "" {
		core.G_secrets["secrets.GITHUB_TOKEN"] = os.Getenv("INPUT_TOKEN")
	} else {
		core.G_secrets["secrets.GITHUB_TOKEN"] = os.Getenv("ACTIONS_RUNTIME_TOKEN")
	}

	err := core.RegisterNodeFactory(ghActionNodeDefinition, func(ctx interface{}) (core.NodeRef, error) {

		nodeType := ctx.(string)

		_, owner, name, ref, err := parseNodeTypeUri(nodeType)
		if err != nil {
			return nil, err
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		actionFolder := filepath.Join(home, "work", "_actions", owner, name, ref)

		// If the action is not already cloned, clone it
		_, err = os.Stat(actionFolder)
		if os.IsNotExist(err) {
			// Clone the entire repo but don't check out yet since HEAD might not be the requested ref.
			cloneUrl := fmt.Sprintf("https://%s@github.com/%s/%s", core.G_githubToken, owner, name)
			c := exec.Command("git", "clone", "--no-checkout", cloneUrl, actionFolder)
			// c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			err = c.Run()
			if err != nil {
				return nil, err
			}

			// run checkout
			c = exec.Command("git", "checkout", u.If(ref == "", "HEAD", ref))
			// c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Dir = actionFolder
			err = c.Run()
			if err != nil {
				return nil, err
			}
		} else {
			// reset in case something tampered with the directory
			c := exec.Command("git", "reset", "--hard", u.If(ref == "", "HEAD", ref))
			// c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Dir = actionFolder
			err = c.Run()
			if err != nil {
				return nil, err
			}
		}

		actionContent, err := os.ReadFile(filepath.Join(actionFolder, "action.yml"))
		if err != nil {
			return nil, errors.New("repo is not a github action")
		}

		var action GithubActionDefinition
		err = yaml.Unmarshal(actionContent, &action)
		if err != nil {
			return nil, err
		}

		node := &GhActionNode{
			actionName: action.Name,
			actionRuns: action.Runs,
		}

		// Potential values for `using`. See the github action runners for more info.
		// https://github.com/actions/runner/blob/a4c57f27477077e57545af79851551ff7f5632bd/src/Runner.Worker/ActionManifestManager.cs#L430-L453
		switch action.Runs.Using {
		case "docker":
			dockerUrl := strings.TrimPrefix(action.Runs.Image, "docker://")
			if dockerUrl == "" {
				return nil, fmt.Errorf("docker image not specified")
			}

			exitCode, err := DockerPull(context.Background(), dockerUrl)
			if err != nil {
				return nil, err
			}
			if exitCode != 0 {
				return nil, fmt.Errorf("docker pull failed with exit code %d", exitCode)
			}

			node.actionType = Docker
		case "node12":
			fallthrough
		case "node14":
			fallthrough
		case "node16":
			fallthrough
		case "node20":
			node.actionType = Node
			actionRunFile := filepath.Join(actionFolder, action.Runs.Main)
			_, err := os.Stat(actionRunFile)
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("action run file does not exist: %s", actionRunFile)
			}

			node.actionRunJsPath = actionRunFile

		case "composite":
			fallthrough
		default:
			return nil, fmt.Errorf("unsupported action run type: %s", action.Runs.Using)
		}

		if len(action.Inputs) > 0 {
			inputs := make(map[core.InputId]core.InputDefinition, 0)

			for name, input := range action.Inputs {
				pd := core.InputDefinition{
					Name:        name,
					Type:        "string",
					Description: input.Description,
				}
				if input.Default != "" {
					pd.Default = input.Default
				}
				inputs[core.InputId(name)] = pd
			}

			node.SetInputDefs(inputs)
		}

		if len(action.Outputs) > 0 {
			outputs := make(map[core.OutputId]core.OutputDefinition, 0)

			for name, output := range action.Outputs {
				outputs[core.OutputId(name)] = core.OutputDefinition{
					Name:        name,
					Type:        "string",
					Description: output.Description,
				}
			}

			node.SetOutputDefs(outputs)
		}

		return node, nil
	})
	if err != nil {
		panic(err)
	}
}

type GithubActionDefinition struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Inputs      map[string]ActionInput  `json:"inputs"`
	Outputs     map[string]ActionOutput `json:"outputs"`
	Runs        ActionRuns              `json:"runs"`
}

type ActionInput struct {
	Default     string `json:"default"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type ActionOutput struct {
	Description string `json:"description"`
}

type ActionRuns struct {
	Image string   `json:"image"`
	Using string   `json:"using"`
	Main  string   `json:"main"`
	Post  string   `json:"post"`
	Args  []string `json:"args"`
}

// parseOutputFile parses the GITHUB_OUTPUT file.
func parseOutputFile(input string) (map[string]string, error) {
	// Example content:
	// ```
	// go-version<<ghadelimiter_22f0e4a6-7c00-4420-bfa4-5b2c88ea9317
	// 1.21.4
	// ghadelimiter_22f0e4a6-7c00-4420-bfa4-5b2c88ea9317
	// go-version<<ghadelimiter_df97302d-20ec-411d-9b03-531a7044b93a
	// 1.21.4
	// ghadelimiter_df97302d-20ec-411d-9b03-531a7044b93a
	// ```

	// This regex captures two groups: the key and the delimiter
	re, err := regexp.Compile(`(.*?)<<ghadelimiter_([a-f0-9\-]+)`)
	if err != nil {
		return nil, err
	}

	matches := re.FindAllStringSubmatch(input, -1)

	kv := make(map[string]string)

	for _, match := range matches {
		if len(match) == 3 {
			key := match[1]
			delimiter := "ghadelimiter_" + match[2]

			// Split the input on the delimiter to get the value
			// go-version<<ghadelimiter_22f0e4a6-7c00-4420-bfa4-5b2c88ea9317
			parts := strings.Split(input, delimiter)
			if len(parts) > 1 {
				value := strings.TrimSpace(parts[1])
				// Split the delimiter, so the value is the first element of the second part
				value = strings.TrimSpace(strings.SplitN(value, delimiter, 2)[0])
				kv[key] = value
			}
		}
	}

	return kv, nil
}

// findHighestVersionDir finds the highest semantic version directory from a given path.
// Used to find the highest runner version as I can't find the runner version anywhere.
func findHighestVersionDir(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	var highestVersion *semver.Version
	var highestVersionDir string

	for _, file := range files {
		if file.IsDir() {
			ver, err := semver.NewVersion(file.Name())
			if err == nil {
				if highestVersion == nil || ver.GreaterThan(highestVersion) {
					highestVersion = ver
					highestVersionDir = file.Name()
				}
			}
		}
	}

	if highestVersion == nil {
		return "", fmt.Errorf("no valid semantic version directories found")
	}

	return highestVersionDir, nil
}

func sanitize(name string, allowHyphens bool) string {
	if name == "" {
		return ""
	}

	var sb strings.Builder
	for i := 0; i < len(name); i++ {
		if (name[i] >= 'a' && name[i] <= 'z') ||
			(name[i] >= 'A' && name[i] <= 'Z') ||
			(name[i] >= '0' && name[i] <= '9' && sb.Len() > 0) ||
			(name[i] == '_') ||
			(allowHyphens && name[i] == '-' && sb.Len() > 0) {
			sb.WriteByte(name[i])
		}
	}
	return sb.String()
}
