//go:build github_impl

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"

	"github.com/google/shlex"
)

var (
	onceGithubVarsRe sync.Once
	githubVarsRe     *regexp.Regexp

	onceNodeTypeUriRegex sync.Once
	nodeTypeUriRegex     *regexp.Regexp
)

var (
	// This is the token of `ACTIONS_RUNTIME_TOKEN` that is used to
	// communicate with certain github APIs, like when github actions
	// are pulled from github.com.
	ghActionsRuntimeToken string

	// This is the map of secrets that are available during the execution
	// of the action graph. The values contain the context name and
	// the secret value. Example: 'secrets.input1' or 'secrets.GITHUB_TOKEN'
	ghSecrets = make(map[string]string, 0)

	// this is the map of 'github.xyz' context variables
	ghContext = make(map[string]string)

	// this is the map of all 'matrix.xyz' variables
	ghMatrix = make(map[string]string)

	// this is the map of all 'inputs.xyz' variables
	ghInputs = make(map[string]string)
)

func AddSecret(name string, secret string) {
	ghSecrets[name] = secret
}

func GetSecret(name string) (string, bool) {
	secret, ok := ghSecrets[name]
	return secret, ok
}

func RemoveSecret(name string) {
	delete(ghSecrets, name)
}

func getGithubVarsRe() *regexp.Regexp {
	onceGithubVarsRe.Do(func() {
		githubVarsRe = regexp.MustCompile(`\$\{\{\s*(env|github|matrix|inputs|secrets)\.[\w]+\s*\}\}`)
	})
	return githubVarsRe
}

func GhActionsRuntimeToken() string {
	return ghActionsRuntimeToken
}

func decodeJsonFromEnvValue(envValue string) (map[string]string, error) {
	envMap := map[string]string{}
	if envValue != "" {
		tmp := map[string]string{}
		err := json.NewDecoder(strings.NewReader(envValue)).Decode(&tmp)
		if err != nil {
			return nil, err
		}
		for k, v := range tmp {
			envMap[k] = v
		}
	}
	return envMap, nil
}

func GetNodeTypeUriRegex() *regexp.Regexp {
	// Information about valid characters in owner, repo and ref names:
	// https://docs.github.com/en/get-started/using-git/dealing-with-special-characters-in-branch-and-tag-names#naming-branches-and-tags
	onceNodeTypeUriRegex.Do(func() {
		nodeTypeUriRegex = regexp.MustCompile(`^(github\.com/)?([-\w]+)/([-\w]+)(@[-\w/\.]+)?`)
	})
	return nodeTypeUriRegex
}

type ContainerInfo struct {
	ContainerDisplayName          string
	ContainerWorkDirectory        string
	ContainerEnvironmentVariables map[string]string
	ContainerEntryPoint           string
	ContainerNetwork              string
	MountVolumes                  []Volume
	ContainerImage                string
	ContainerEntryPointArgs       string
}

type Volume struct {
	SourceVolumePath string
	TargetVolumePath string
	ReadOnly         bool
}

func SplitAtCommas(s string) []string {
	var res []string
	var beg int
	var inString bool

	for i, char := range s {
		switch {
		case char == ',' && !inString:
			res = append(res, s[beg:i])
			beg = i + 1
		case char == '"':
			inString = !inString || (i > 0 && s[i-1] != '\\')
		}
	}

	return append(res, s[beg:])
}

func ExecuteDockerCommand(ctx context.Context, command string, optionsString string, workdir string, stdoutDataReceived chan string, stderrDataReceived chan string) (int, error) {
	args, err := shlex.Split(optionsString)
	if err != nil {
		return 1, err
	}
	cmdArgs := append([]string{command}, args...)

	cmd := exec.Command("docker", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = workdir
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if ok {
			exitCode = exitError.ExitCode()
		}
	}

	return exitCode, err
}

func CreateEscapedOption(flag, key, value string) string {
	if key == "" {
		return ""
	}
	escapedString := SanitizeOptionKeyValue(key + "=" + value)
	return flag + " " + escapedString
}

func SanitizeOptionKeyValue(value string) string {
	if value == "" {
		return ""
	}

	pair := strings.SplitN(value, "=", 2)
	if len(pair) == 1 {
		return fmt.Sprintf("%q=", pair[0])
	}

	// If the value contains spaces or quotes, wrap it in quotes
	if strings.ContainsAny(value, " \t\"") {
		return fmt.Sprintf("%s=%q", pair[0], strings.ReplaceAll(pair[1], "\"", "\\\""))
	}
	return value
}

func DockerRun(ctx context.Context, label string, container ContainerInfo, workingDirectory string, stdoutDataReceived, stderrDataReceived chan string) (int, error) {
	var dockerOptions []string

	dockerOptions = append(dockerOptions,
		fmt.Sprintf("--name %s", container.ContainerDisplayName),
		fmt.Sprintf("--label %s", "actionforge"),
		fmt.Sprintf("--workdir %s", container.ContainerWorkDirectory),
		"--rm",
	)

	for key, value := range container.ContainerEnvironmentVariables {
		dockerOptions = append(dockerOptions, CreateEscapedOption("-e", key, value))
	}

	dockerOptions = append(dockerOptions, "-e GITHUB_ACTIONS=true")

	if _, exists := container.ContainerEnvironmentVariables["CI"]; !exists {
		dockerOptions = append(dockerOptions, "-e CI=true")
	}

	if container.ContainerEntryPoint != "" {
		dockerOptions = append(dockerOptions, fmt.Sprintf("--entrypoint \"%s\"", container.ContainerEntryPoint))
	}

	if container.ContainerNetwork != "" {
		dockerOptions = append(dockerOptions, fmt.Sprintf("--network %s", container.ContainerNetwork))
	}

	for _, volume := range container.MountVolumes {
		mountArg := formatMountArg(volume)
		dockerOptions = append(dockerOptions, mountArg)
	}

	dockerOptions = append(dockerOptions, container.ContainerImage)
	dockerOptions = append(dockerOptions, container.ContainerEntryPointArgs)

	optionsString := strings.Join(dockerOptions, " ")
	return ExecuteDockerCommand(ctx, "run", optionsString, workingDirectory, stdoutDataReceived, stderrDataReceived)
}

func formatMountArg(volume Volume) string {
	var volumeArg string
	if volume.SourceVolumePath == "" {
		volumeArg = fmt.Sprintf("-v \"%s\"", escapePath(volume.TargetVolumePath))
	} else {
		volumeArg = fmt.Sprintf("-v \"%s\":\"%s\"", escapePath(volume.SourceVolumePath), escapePath(volume.TargetVolumePath))
	}
	if volume.ReadOnly {
		volumeArg += ":ro"
	}
	return volumeArg
}

func escapePath(path string) string {
	return strings.ReplaceAll(path, "\"", "\\\"")
}

func DockerPull(ctx context.Context, image string, workingDirectory string) (int, error) {

	LoggerBase.Printf("%sPull down action image '%s'.\n",
		LogGhStartGroup,
		image,
	)

	defer LoggerBase.Printf(LogGhEndGroup)

	return ExecuteDockerCommand(ctx, "pull", image, workingDirectory, nil, nil)
}

func DockerBuild(ctx context.Context, workingDirectory string, dockerFile string, dockerContext string, tag string) (int, error) {
	buildOptions := fmt.Sprintf("-t %s -f \"%s\" \"%s\"", tag, dockerFile, dockerContext)
	return ExecuteDockerCommand(ctx, "build", buildOptions, workingDirectory, nil, nil)
}

func ReplaceContextVariables(input string) string {

	return getGithubVarsRe().ReplaceAllStringFunc(input, func(s string) string {
		// Remove the template syntax to get the context variable
		contextVar := strings.Trim(s, "${ }")

		if strings.HasPrefix(contextVar, "github.") {
			envVar, exists := ghContext[contextVar]
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "env.") {
			envVar, exists := os.LookupEnv(strings.TrimPrefix(contextVar, "env."))
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "secrets.") {
			secretVal, exists := ghSecrets[strings.TrimPrefix(contextVar, "secrets.")]
			if exists {
				return secretVal
			}
			return ""
		} else if strings.HasPrefix(contextVar, "matrix.") {
			envVar, exists := ghMatrix[strings.TrimPrefix(contextVar, "matrix.")]
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "inputs.") {
			envVar, exists := ghInputs[strings.TrimPrefix(contextVar, "inputs.")]
			if exists {
				return envVar
			}
			return ""
		}

		// If the context variable is not found, return the original string
		// Should never happen as the regex should only match the context variables above.
		return s
	})
}
func InitGhContexts() error {

	// For more information on the githubs context, see:
	// https://docs.github.com/en/actions/learn-github-actions/contexts

	ghContext["github.action"] = os.Getenv("GITHUB_ACTION")
	// No direct mapping yet for 'github.action_path'
	ghContext["github.actor"] = os.Getenv("GITHUB_ACTOR")
	ghContext["github.actor_id"] = os.Getenv("GITHUB_ACTOR_ID")
	ghContext["github.api_url"] = os.Getenv("GITHUB_API_URL")
	ghContext["github.base_ref"] = os.Getenv("GITHUB_BASE_REF")
	ghContext["github.env"] = os.Getenv("GITHUB_ENV")
	ghContext["github.event_name"] = os.Getenv("GITHUB_EVENT_NAME")
	ghContext["github.event_path"] = os.Getenv("GITHUB_EVENT_PATH")
	ghContext["github.graphql_url"] = os.Getenv("GITHUB_GRAPHQL_URL")
	ghContext["github.head_ref"] = os.Getenv("GITHUB_HEAD_REF")
	ghContext["github.job"] = os.Getenv("GITHUB_JOB")
	ghContext["github.ref"] = os.Getenv("GITHUB_REF")
	ghContext["github.ref_name"] = os.Getenv("GITHUB_REF_NAME")
	ghContext["github.ref_protected"] = os.Getenv("GITHUB_REF_PROTECTED")
	ghContext["github.ref_type"] = os.Getenv("GITHUB_REF_TYPE")
	ghContext["github.repository"] = os.Getenv("GITHUB_REPOSITORY")
	ghContext["github.repository_id"] = os.Getenv("GITHUB_REPOSITORY_ID")
	ghContext["github.repository_owner"] = os.Getenv("GITHUB_REPOSITORY_OWNER")
	ghContext["github.repository_owner_id"] = os.Getenv("GITHUB_REPOSITORY_OWNER_ID")
	ghContext["github.run_attempt"] = os.Getenv("GITHUB_RUN_ATTEMPT")
	ghContext["github.run_id"] = os.Getenv("GITHUB_RUN_ID")
	ghContext["github.run_number"] = os.Getenv("GITHUB_RUN_NUMBER")
	ghContext["github.server_url"] = os.Getenv("GITHUB_SERVER_URL")
	ghContext["github.sha"] = os.Getenv("GITHUB_SHA")
	ghContext["github.workflow"] = os.Getenv("GITHUB_WORKFLOW")
	ghContext["github.workflow_ref"] = os.Getenv("GITHUB_WORKFLOW_REF")
	ghContext["github.workspace"] = os.Getenv("GITHUB_WORKSPACE")

	// As outlined in the documentation, secrets.GITHUB_TOKEN and github.token
	// are functionally equivalent. See:
	// https://docs.github.com/en/actions/learn-github-actions/contexts#github-context
	ghContext["github.token"] = os.Getenv("INPUT_TOKEN")
	ghSecrets["GITHUB_TOKEN"] = os.Getenv("INPUT_TOKEN")

	ghActionsRuntimeToken = os.Getenv("ACTIONS_RUNTIME_TOKEN")

	for _, env := range os.Environ() {

		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			continue
		}

		envName := kv[0]
		envValue := kv[1]

		if envName == "INPUT_MATRIX" {
			var err error
			ghMatrix, err = decodeJsonFromEnvValue(envValue)
			if err != nil {
				return err
			}
			os.Unsetenv(envName)
		} else if envName == "INPUT_INPUTS" {
			var err error
			ghInputs, err = decodeJsonFromEnvValue(envValue)
			if err != nil {
				return err
			}
			os.Unsetenv(envName)
		} else if envName == "INPUT_SECRETS" {
			secrets, err := decodeJsonFromEnvValue(envValue)
			if err != nil {
				return err
			}
			for k, v := range secrets {
				ghSecrets[k] = v
			}
			os.Unsetenv(envName)
		} else if strings.HasPrefix(envName, "SECRET_") {

			/* Deprecated, replaced by INPUT_SECRETS */
			k := strings.TrimPrefix(envName, "SECRET_")
			if k != "" {
				ghSecrets[k] = envValue
			}
			os.Unsetenv(envName)
		}
	}

	// The information in the inputs context and github.event.inputs context is identical
	// except that the inputs context preserves Boolean values as Booleans instead of converting
	// them to strings. TODO: (Seb) Change the ghInputs to map[string]interface{} to preserve
	// the types for inputs.
	for k, v := range ghSecrets {
		ghContext[fmt.Sprintf("github.event.%s", k)] = v
	}

	return nil
}

func init() {
	LoadEnvOnce()

	err := InitGhContexts()
	if err != nil {
		log.Fatalf("Error initializing github context: %s", err)
	}
}
