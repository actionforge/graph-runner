//go:build github_impl

package nodes

import (
	u "actionforge/graph-runner/utils"
	"context"
	"fmt"
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

func getGithubVarsRe() *regexp.Regexp {
	onceGithubVarsRe.Do(func() {
		githubVarsRe = regexp.MustCompile(`\$\{\{\s*(env|github|matrix|inputs|secrets)\.[\w]+\s*\}\}`)
	})
	return githubVarsRe
}

func getNodeTypeUriRegex() *regexp.Regexp {
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

	u.LoggerBase.Printf("%sPull down action image '%s'.\n",
		u.LogGhStartGroup,
		image,
	)

	defer u.LoggerBase.Printf(u.LogGhEndGroup)

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
			envVar, exists := ghSecrets[contextVar]
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "matrix.") {
			envVar, exists := ghMatrix[contextVar]
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "inputs.") {
			envVar, exists := ghInputs[contextVar]
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
