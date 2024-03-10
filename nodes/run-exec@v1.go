package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

//go:embed run-exec@v1.yml
var execDefinition string

type RunExecNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *RunExecNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	path, err := core.InputValueById[string](c, n.Inputs, ni.Run_exec_v1_Input_path)
	if err != nil {
		return err
	}

	print, err := core.InputValueById[string](c, n.Inputs, ni.Run_exec_v1_Input_print)
	if err != nil {
		return err
	}

	args, err := core.InputValueById[[]string](c, n.Inputs, ni.Run_exec_v1_Input_args)
	if err != nil {
		return err
	}

	envs, err := core.InputValueById[[]string](c, n.Inputs, ni.Run_exec_v1_Input_env)
	if err != nil {
		return err
	}

	stdin, err := core.InputValueById[string](c, n.Inputs, ni.Run_exec_v1_Input_stdin)
	if err != nil {
		return err
	}

	for i, env := range envs {
		envs[i] = ReplaceContextVariables(env)
	}

	env := append(envs, os.Environ()...)

	cmd := exec.Command(path, args...)
	cmd.Env = env

	var (
		output []byte
		cmdErr error
	)

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	if print == "stdout" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmdErr = cmd.Run()
	} else if print == "output" || print == "both" {
		output, cmdErr = cmd.CombinedOutput()
		var decoder *encoding.Decoder
		if runtime.GOOS == "windows" {
			if utils.IsUtf16Le(output) {
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

	err = n.SetOutputValue(c, ni.Run_exec_v1_Output_output, string(output))
	if err != nil {
		return err
	}

	err = n.SetOutputValue(c, ni.Run_exec_v1_Output_exit_code, cmd.ProcessState.ExitCode())
	if err != nil {
		return err
	}

	if cmd.ProcessState.ExitCode() == 0 {
		err = n.Execute(ni.Run_exec_v1_Output_exec_success, c)
		if err != nil {
			return err
		}
	} else {
		_, ok := n.GetExecutionTarget(ni.Run_exec_v1_Output_exec_err)
		// If the error output is not connected, we can safely fail here
		if !ok {
			return utils.Throw(cmdErr)
		}

		err = n.Execute(ni.Run_exec_v1_Output_exec_err, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(execDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &RunExecNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
