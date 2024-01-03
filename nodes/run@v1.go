package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
	"errors"
	"os"
	"os/exec"
	"runtime"
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

	for i, env := range envs {
		envs[i] = ReplaceContextVariables(env, n.GetInputValues())
	}

	env := append(envs, os.Environ()...)

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
	cmd.Env = env

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

	err = n.SetOutputValue(c, ni.Run_v1_Output_output, string(output))
	if err != nil {
		return err
	}

	err = n.SetOutputValue(c, ni.Run_v1_Output_exit_code, cmd.ProcessState.ExitCode())
	if err != nil {
		return err
	}

	if cmd.ProcessState.ExitCode() == 0 {
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
