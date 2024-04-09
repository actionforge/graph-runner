//go:build github_impl

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
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

var (
	dockerGithubWorkspace    = "/github/workspace"
	dockerGithubWorkflow     = "/github/workflow"
	dockerGithubFileCommands = "/github/file_commands"
	dockerGithubHome         = "/github/home"

	//go:embed gh-action@v1.yml
	ghActionNodeDefinition string
)

type ActionType int

const (
	Docker ActionType = iota
	Node
)

type DockerData struct {
	Image               string
	DockerInstanceLabel string
	ExecutionContextId  string
}

type GhActionNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	actionName      string
	actionType      ActionType // docker or node
	actionRuns      ActionRuns
	actionRunJsPath string

	Data DockerData
}

type EnvironArgs struct {
	ExecutionEnviron map[string]string
	CustomEnvs       map[string]bool
}

func (n *GhActionNode) ExecuteImpl(c core.ExecutionContext) error {
	contextEnvironMap := c.GetContextEnvironMapCopy()

	sysRunnerTempDir := contextEnvironMap["RUNNER_TEMP"]
	if sysRunnerTempDir == "" {
		return fmt.Errorf("RUNNER_TEMP not set")
	}

	sysGhWorkspaceDir := contextEnvironMap["GITHUB_WORKSPACE"]
	if sysGhWorkspaceDir == "" {
		return fmt.Errorf("GITHUB_WORKSPACE not set")
	}

	runnerToolCache := contextEnvironMap["RUNNER_TOOL_CACHE"]
	if runnerToolCache == "" {
		return fmt.Errorf("RUNNER_TOOL_CACHE not set")
	}

	withInputs := ""

	for inputName := range n.Inputs.GetInputDefs() {
		v, err := core.InputValueById[string](c, n.Inputs, inputName)
		if err != nil {
			return u.Throw(err)
		}
		v = ReplaceContextVariables(v)
		contextEnvironMap[fmt.Sprintf("INPUT_%v", strings.ToUpper(string(inputName)))] = v
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
	envs, err := core.InputValueById[[]string](c, n.Inputs, "env")
	if err != nil {
		_, ok := err.(*u.ErrNoInputValue)
		if !ok {
			return u.Throw(err)
		}
	}

	customEnvs := map[string]bool{}

	for _, env := range envs {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			customEnvs[kv[0]] = true
			contextEnvironMap[kv[0]] = ReplaceContextVariables(kv[1])
		}
	}

	ghContextParser := GhContextParser{}
	ctxEnvs, err := ghContextParser.Init(sysRunnerTempDir)
	if err != nil {
		return u.Throw(err)
	}
	for envName, path := range ctxEnvs {
		// Set GITHUB_PATH, GITHUB_ENV, etc.
		contextEnvironMap[envName] = path
	}

	// https://github.com/actions/runner/blob/f467e9e1255530d3bf2e33f580d041925ab01951/src/Runner.Common/HostContext.cs#L288
	if contextEnvironMap["AGENT_TOOLSDIRECTORY"] == "" {
		contextEnvironMap["AGENT_TOOLSDIRECTORY"] = contextEnvironMap["RUNNER_TOOL_CACHE"]
	}
	if contextEnvironMap["RUNNER_TOOLSDIRECTORY"] == "" {
		contextEnvironMap["RUNNER_TOOLSDIRECTORY"] = contextEnvironMap["RUNNER_TOOL_CACHE"]
	}

	if n.actionType == Docker {
		err = n.ExecuteDocker(c, sysGhWorkspaceDir, EnvironArgs{
			ExecutionEnviron: contextEnvironMap,
			CustomEnvs:       customEnvs,
		})
	} else if n.actionType == Node {
		err = n.ExecuteNode(c, sysGhWorkspaceDir, EnvironArgs{
			ExecutionEnviron: contextEnvironMap,
		})
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
			return u.Throw(err)
		}
	}

	// Set the output values to empty strings. If an action didn't set an output,
	// it will evaluate to an empty string in a subsequent action if result wasn't set.
	for outputId := range n.OutputDefsCopy() {
		err = n.SetOutputValue(c, outputId, "")
		if err != nil {
			return u.Throw(err)
		}
	}

	// Get the context vars from GITHUB_ENV and GITHUB_PATH
	ctxEnvs, err = ghContextParser.Parse(contextEnvironMap)
	if err != nil {
		return u.Throw(err)
	}
	for envName, envValue := range ctxEnvs {
		contextEnvironMap[envName] = envValue
	}

	// Transfer the output values from the github action to the node output values
	githubOutput := contextEnvironMap["GITHUB_OUTPUT"]
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
			err = n.SetOutputValue(c, core.OutputId(key), strings.TrimRight(value, "\t\n"))
			if err != nil {
				return u.Throw(err)
			}
		}

		err = os.Remove(githubOutput)
		if err != nil {
			return u.Throw(err)
		}
	}

	c.SetContextEnvironMap(contextEnvironMap)

	err = n.Execute(n.Executions[ni.Gh_action_v1_Output_exec], c)
	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func (n *GhActionNode) ExecuteNode(c core.ExecutionContext, workspace string, envs EnvironArgs) error {

	nodeBin := "node"
	runners, err := getRunnersDir()
	if err == nil {
		externalNodeBin := filepath.Join(runners, "externals", n.actionRuns.Using, "bin", "node")
		_, err := os.Stat(nodeBin)
		if err == nil {
			nodeBin = externalNodeBin
		}
	}

	fmt.Printf("Use node binary: %s\n", nodeBin)

	cmd := exec.Command(nodeBin, n.actionRunJsPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = nil
	cmd.Env = func() []string {
		env := make([]string, 0)
		for k, v := range envs.ExecutionEnviron {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		return env
	}()
	cmd.Dir = workspace
	err = cmd.Run()
	if err != nil {
		return u.Throw(err)
	}
	return nil
}

func (n *GhActionNode) ExecuteDocker(c core.ExecutionContext, workingDirectory string, envs EnvironArgs) error {
	sysRunnerTempDir := envs.ExecutionEnviron["RUNNER_TEMP"]
	if sysRunnerTempDir == "" {
		return u.Throw(fmt.Errorf("RUNNER_TEMP not set"))
	}

	sysGithubWorkspace := envs.ExecutionEnviron["GITHUB_WORKSPACE"]
	if sysGithubWorkspace == "" {
		return u.Throw(fmt.Errorf("GITHUB_WORKSPACE not set"))
	}

	if envs.CustomEnvs == nil {
		envs.CustomEnvs = make(map[string]bool)
	}

	// Only allow certain environment variables to be passed to the docker container
	dockerEnviron := make(map[string]string)
	for contextName := range contextEnvAllowList {
		envName := fmt.Sprintf("GITHUB_%s", strings.ToUpper(contextName))
		dockerEnviron[envName] = envs.ExecutionEnviron[envName]
	}

	for k, v := range envs.ExecutionEnviron {
		if strings.HasPrefix(k, "RUNNER_") || strings.HasPrefix(k, "INPUT_") || envs.CustomEnvs[k] {
			dockerEnviron[k] = v
		}
	}

	// Update github context paths to the docker paths
	for _, envName := range contextEnvList {
		path := envs.ExecutionEnviron[envName]
		if path == "" {
			return u.Throw(fmt.Errorf("expected %s to be set in execution environment", envName))
		}
		dockerEnviron[envName] = filepath.Join(dockerGithubFileCommands, filepath.Base(path))
	}

	// Set new env vars for container
	dockerEnviron["HOME"] = dockerGithubHome

	ContainerEntryArgs := make([]string, 0)
	for _, arg := range n.actionRuns.Args {
		ContainerEntryArgs = append(ContainerEntryArgs, ReplaceContextVariables(arg))
	}

	ci := ContainerInfo{
		ContainerImage:                n.Data.Image,
		ContainerDisplayName:          fmt.Sprintf("actionforge_%s_%s", n.Data.DockerInstanceLabel, uuid.New()),
		ContainerWorkDirectory:        dockerGithubWorkspace,
		ContainerEntryPointArgs:       strings.Join(ContainerEntryArgs, " "),
		ContainerEnvironmentVariables: dockerEnviron,
	}

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		ci.MountVolumes = append(ci.MountVolumes, Volume{
			SourceVolumePath: "/var/run/docker.sock",
			TargetVolumePath: "/var/run/docker.sock",
			ReadOnly:         false,
		})
	}

	// All mounted volumes from the original runner are registered here:
	// https://github.com/actions/runner/blob/f467e9e1255530d3bf2e33f580d041925ab01951/src/Runner.Worker/Handlers/ContainerActionHandler.cs#L193-L197

	ci.MountVolumes = append(ci.MountVolumes, Volume{
		SourceVolumePath: sysGithubWorkspace,
		TargetVolumePath: dockerGithubWorkspace,
		ReadOnly:         false,
	})

	ci.MountVolumes = append(ci.MountVolumes, Volume{
		SourceVolumePath: filepath.Join(sysRunnerTempDir, "_github_workflow"),
		TargetVolumePath: dockerGithubWorkflow,
		ReadOnly:         false,
	})

	ci.MountVolumes = append(ci.MountVolumes, Volume{
		SourceVolumePath: filepath.Join(sysRunnerTempDir, "_github_home"),
		TargetVolumePath: dockerGithubHome,
		ReadOnly:         false,
	})

	ci.MountVolumes = append(ci.MountVolumes, Volume{
		SourceVolumePath: filepath.Join(sysRunnerTempDir, "_runner_file_commands"),
		TargetVolumePath: dockerGithubFileCommands,
		ReadOnly:         false,
	})

	exitCode, err := DockerRun(context.Background(), n.Data.DockerInstanceLabel, ci, workingDirectory, nil, nil)
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
			cloneUrl := fmt.Sprintf("https://%s@github.com/%s/%s", ghActionsRuntimeToken, owner, name)
			c := exec.Command("git", "clone", "--quiet", "--no-checkout", cloneUrl, actionFolder)
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
			c := exec.Command("git", "reset", "--quiet", "--hard", u.If(ref == "", "HEAD", ref))
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
			sysWorkspaceDir := os.Getenv("GITHUB_WORKSPACE")
			if sysWorkspaceDir == "" {
				return nil, fmt.Errorf("GITHUB_WORKSPACE not set")
			}

			node.actionType = Docker

			// The Docker image to use as the container to run the action.
			// The value can be the Docker Hub image name or a registry name.
			if strings.HasPrefix(action.Runs.Image, "docker://") {
				dockerUrl := strings.TrimPrefix(action.Runs.Image, "docker://")
				if dockerUrl == "" {
					return nil, fmt.Errorf("docker image not specified")
				}

				node.Data.Image = dockerUrl
				exitCode, err := DockerPull(context.Background(), dockerUrl, sysWorkspaceDir)
				if err != nil {
					return nil, err
				}
				if exitCode != 0 {
					return nil, fmt.Errorf("docker pull failed with exit code %d", exitCode)
				}
			} else {
				// TODO: (Seb) DockerInstanceLabel is part of the original actions/runner implementation.
				// It's a sha256 of a json within the <version>/.runner directory.
				// DockerInstanceLabel
				executionContextId := uuid.New()

				runnersDir, err := getRunnersDir()
				if err != nil {
					return nil, err
				}

				// https://github.com/actions/runner/blob/77e0bfbb8a8fde1f01fc1cf1ed2d7f0e81a0a407/src/Runner.Worker/Container/DockerCommandManager.cs#L48-L52
				runnersSha256, err := u.GetSha256OfFile(filepath.Join(runnersDir, ".runner"))
				if err != nil {
					return nil, err
				}
				node.Data.DockerInstanceLabel = runnersSha256[:6]

				// https://github.com/actions/runner/blob/77e0bfbb8a8fde1f01fc1cf1ed2d7f0e81a0a407/src/Runner.Worker/Handlers/ContainerActionHandler.cs#L68
				imageName := fmt.Sprintf("%s:%s", node.Data.DockerInstanceLabel, executionContextId.String())
				node.Data.Image = imageName
				node.Data.ExecutionContextId = executionContextId.String()

				u.LoggerBase.Printf("%sBuild container for action use '%s'.\n",
					u.LogGhStartGroup,
					"",
				)

				exitCode, err := DockerBuild(context.Background(), actionFolder, path.Join(actionFolder, action.Runs.Image), actionFolder, imageName)
				if err != nil {
					return nil, err
				}

				if exitCode != 0 {
					return nil, fmt.Errorf("docker build failed with exit code %d", exitCode)
				}

				u.LoggerBase.Printf(u.LogGhEndGroup)
			}
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

		node.SetNodeType(nodeType)
		node.SetName(action.Name)
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

// getRunnersDir returns the directory of the latest runner version.
func getRunnersDir() (string, error) {

	// TODO: (Seb) This function iterates over the different runner versions
	// in the home folder to find the latest dir version. There is currently
	// no other way to find the real runner version.

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	files, err := os.ReadDir(filepath.Join(homeDir, "runners"))
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

	return filepath.Join(homeDir, "runners", highestVersionDir), nil
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

// https://github.com/actions/runner/blob/f467e9e1255530d3bf2e33f580d041925ab01951/src/Runner.Worker/GitHubContext.cs#L9
var contextEnvAllowList = map[string]struct{}{
	"action_path":         {},
	"action_ref":          {},
	"action_repository":   {},
	"action":              {},
	"actor":               {},
	"actor_id":            {},
	"api_url":             {},
	"base_ref":            {},
	"env":                 {},
	"event_name":          {},
	"event_path":          {},
	"graphql_url":         {},
	"head_ref":            {},
	"job":                 {},
	"output":              {},
	"path":                {},
	"ref_name":            {},
	"ref_protected":       {},
	"ref_type":            {},
	"ref":                 {},
	"repository":          {},
	"repository_id":       {},
	"repository_owner":    {},
	"repository_owner_id": {},
	"retention_days":      {},
	"run_attempt":         {},
	"run_id":              {},
	"run_number":          {},
	"server_url":          {},
	"sha":                 {},
	"state":               {},
	"step_summary":        {},
	"triggering_actor":    {},
	"workflow":            {},
	"workflow_ref":        {},
	"workflow_sha":        {},
	"workspace":           {},
}
