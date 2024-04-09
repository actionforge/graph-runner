package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

//go:embed run@v1.yml
var runDefinition string

var (
	pythonName     string
	oncePythonName *sync.Once
)

type RunNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *RunNode) ExecuteImpl(c core.ExecutionContext) error {
	shell, err := core.InputValueById[string](c, n.Inputs, ni.Run_v1_Input_shell)
	if err != nil {
		return err
	}

	script, err := core.InputValueById[string](c, n.Inputs, ni.Run_v1_Input_script)
	if err != nil {
		return err
	}

	print, err := core.InputValueById[string](c, n.Inputs, ni.Run_v1_Input_print)
	if err != nil {
		return err
	}

	envs, err := core.InputValueById[[]string](c, n.Inputs, ni.Run_v1_Input_env)
	if err != nil {
		return err
	}

	contextEnvironMap := c.GetContextEnvironMapCopy()
	for _, env := range envs {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			contextEnvironMap[kv[0]] = ReplaceContextVariables(kv[1])
		}
	}

	ghContextParser := GhContextParser{}
	if contextEnvironMap["GITHUB_ACTIONS"] != "" {
		sysRunnerTempDir := contextEnvironMap["RUNNER_TEMP"]
		if sysRunnerTempDir == "" {
			return fmt.Errorf("RUNNER_TEMP not set")
		}
		ctxEnvs, err := ghContextParser.Init(sysRunnerTempDir)
		if err != nil {
			return u.Throw(err)
		}
		for envName, path := range ctxEnvs {
			// Set GITHUB_PATH, GITHUB_ENV, etc.
			contextEnvironMap[envName] = path
		}
	}

	tmpfilePath := "run-script-*"
	if runtime.GOOS == "windows" {
		switch shell {
		case "cmd":
			tmpfilePath += ".cmd"
		case "pwsh":
			tmpfilePath += ".ps1"
		}
	}

	tmpfile, err := os.CreateTemp("", tmpfilePath)
	if err != nil {
		return err
	}
	defer func() {
		if tmpfile != nil {
			tmpfile.Close()
		}
	}()
	defer os.Remove(tmpfile.Name())

	tmpfileName := tmpfile.Name()

	_, err = tmpfile.WriteString(script)
	if err != nil {
		return err
	}

	err = tmpfile.Close()
	if err != nil {
		return err
	}
	tmpfile = nil

	var cmdArgs []string

	// Calls and their arguments are defined by the GH docs.
	// https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#jobsjob_idstepsshell
	switch shell {
	case "bash":
		cmdArgs = append(cmdArgs, "--noprofile", "--norc", "-eo", "pipefail", tmpfileName)
	case "pwsh":
		cmdArgs = append(cmdArgs, "-command", tmpfileName)
	case "cmd":
		if runtime.GOOS != "windows" {
			return errors.New("cmd shell is only available on Windows")
		}
		cmdPath, ok := os.LookupEnv("ComSpec")
		if ok {
			shell = cmdPath
		}
		cmdArgs = append(cmdArgs, "/D", "/E:ON", "/V:OFF", "/S", "/C", tmpfileName)
	case "python":
		if pythonName == "" {
			oncePythonName = &sync.Once{}
			oncePythonName.Do(func() {

				err := exec.Command("python3", "--version").Run()
				if err == nil {
					pythonName = "python3"
				} else {
					err = exec.Command("python", "--version").Run()
					if err == nil {
						pythonName = "python"
					}
				}
			})
		}

		if pythonName == "" {
			return errors.New("python is not installed")
		}

		shell = pythonName
		cmdArgs = append(cmdArgs, "-c", tmpfileName)
	}

	cmd := exec.Command(shell)
	cmd.Args = cmdArgs
	cmd.Env = func() []string {
		env := make([]string, 0, len(contextEnvironMap))
		for k, v := range contextEnvironMap {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		return env
	}()

	var (
		output []byte
		cmdErr error
	)

	if print == "stdout" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	} else if print == "output" || print == "both" {
		output, cmdErr = cmd.CombinedOutput()
		var decoder *encoding.Decoder
		if runtime.GOOS == "windows" {
			if isUTF16LE(output) {
				// Sometimes Windows returns UTF16
				decoder = unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()
			} else {
				decoder = unicode.UTF8.NewDecoder()
			}
		} else {
			decoder = unicode.UTF8.NewDecoder()
		}
		output, _, err = transform.Bytes(decoder, output)
		if err != nil {
			return err
		}
		utils.LoggerBase.Printf("%s", string(output))
	}
	// cmdErr is processed further down below

	err = n.SetOutputValue(c, ni.Run_v1_Output_output, string(output), core.SetOutputValueOpts{})
	if err != nil {
		return err
	}

	err = n.SetOutputValue(c, ni.Run_v1_Output_exit_code, cmd.ProcessState.ExitCode(), core.SetOutputValueOpts{})
	if err != nil {
		return err
	}

	if cmd.ProcessState.ExitCode() == 0 {

		if contextEnvironMap["GITHUB_ACTIONS"] != "" {
			// Get the context vars from GITHUB_ENV and GITHUB_PATH
			ctxEnvs, err := ghContextParser.Parse(contextEnvironMap)
			if err != nil {
				return u.Throw(err)
			}
			for envName, envValue := range ctxEnvs {
				contextEnvironMap[envName] = envValue
			}
			c.SetContextEnvironMap(contextEnvironMap)
		}

		err = n.Execute(n.Executions[ni.Run_v1_Output_exec_success], c)
		if err != nil {
			return err
		}
	} else {
		execErr := n.Executions[ni.Run_v1_Output_exec_err]

		// If the error output is not connected, we can safely fail here
		if execErr == nil {
			return utils.Throw(cmdErr)
		}

		err = n.Execute(execErr, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func isUTF16LE(b []byte) bool {
	if len(b)%2 != 0 {
		// UTF-16 should have an even number of bytes
		return false
	}

	for i := 0; i < len(b); i += 2 {
		if b[i+1] != 0 {
			return false
		}
	}
	return true
}

func init() {
	err := core.RegisterNodeFactory(runDefinition, func(context interface{}) (core.NodeRef, error) {
		return &RunNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
