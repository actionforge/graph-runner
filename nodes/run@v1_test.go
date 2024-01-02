//go:build integration_tests

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"runtime"
	"strings"
	"testing"
)

func createRunNode(t *testing.T) (*core.NodeRef, core.NodeExecutionInterface, core.HasInputsInterface, core.HasOuputsInterface) {

	r, err := core.NewNodeInstance("run@v1")
	if err != nil {
		t.Fatal(err)
	}

	nei, ok := r.(core.NodeExecutionInterface)
	if !ok {
		t.Error("run@v1 is not a node execution interface")
	}

	inputs, ok := r.(core.HasInputsInterface)
	if !ok {
		t.Fatal("run@v1 has no inputs")
	}

	outputs, ok := r.(core.HasOuputsInterface)
	if !ok {
		t.Fatal("run@v1 has no outputs")
	}

	return &r, nei, inputs, outputs
}

// Run a simple script/program and check that the output is correct.
func Test_RunNode_Simple(t *testing.T) {

	_, nei, inputs, outputs := createRunNode(t)

	inputs.SetInputValue(ni.Run_v1_Input_shell, "python")
	inputs.SetInputValue(ni.Run_v1_Input_script, "import sys;sys.stdout.write('hello world')")

	c := core.EmptyExecutionContext()

	err := nei.ExecuteImpl(c)
	if err != nil {
		t.Fatal(err)
	}

	output, err := outputs.OutputValueById(c, ni.Run_v1_Output_output)
	if err != nil {
		t.Fatal(err)
	}

	if output != "hello world" {
		t.Error("expected output to be 'hello world', got", output)
	}

	exit_code, err := outputs.OutputValueById(c, ni.Run_v1_Output_exit_code)
	if err != nil {
		t.Fatal(err)
	}

	if exit_code != 0 {
		t.Error("expected exit code to be 0, got", exit_code)
	}
}

// Test that environment variables are properly passed to the script/program.
func Test_RunNode_Env(t *testing.T) {

	_, nei, inputs, outputs := createRunNode(t)

	if runtime.GOOS == "windows" {
		inputs.SetInputValue(ni.Run_v1_Input_shell, "python")
		inputs.SetInputValue(ni.Run_v1_Input_env, []string{"MY_VAR=foo bar"})
		inputs.SetInputValue(ni.Run_v1_Input_script, "import os,sys;sys.stdout.write(os.environ['MY_VAR'])")
	} else {
		inputs.SetInputValue(ni.Run_v1_Input_shell, "bash")
		inputs.SetInputValue(ni.Run_v1_Input_env, []string{"MY_VAR=foo bar"})
		inputs.SetInputValue(ni.Run_v1_Input_script, "echo -n $MY_VAR")
	}

	c := core.EmptyExecutionContext()

	err := nei.ExecuteImpl(c)
	if err != nil {
		t.Fatal(err)
	}

	output, err := outputs.OutputValueById(c, ni.Run_v1_Output_output)
	if err != nil {
		t.Fatal(err)
	}

	if output != "foo bar" {
		t.Error("expected output to be 'foo bar', got", output)
	}

	exit_code, err := outputs.OutputValueById(c, ni.Run_v1_Output_exit_code)
	if err != nil {
		t.Fatal(err)
	}

	if exit_code != 0 {
		t.Error("expected exit code to be 0, got", exit_code)
		return
	}
}

// Test that 'exit_code' is properly set when the script/program fails.
func Test_RunNode_ExitCode(t *testing.T) {

	_, nei, inputs, outputs := createRunNode(t)

	inputs.SetInputValue(ni.Run_v1_Input_shell, "python")
	inputs.SetInputValue(ni.Run_v1_Input_script, "abc")

	c := core.EmptyExecutionContext()

	err := nei.ExecuteImpl(c)
	if err == nil {
		t.Error("expected error")
		return
	}

	output, err := outputs.OutputValueById(c, ni.Run_v1_Output_output)
	if err != nil {
		t.Fatal(err)
	}

	o, ok := output.(string)
	if !ok {
		t.Error("expected output to be a string")
		return
	}

	if !strings.Contains(o, "NameError: name 'abc' is not defined") {
		t.Error("expected output to be 'NameError: name 'abc' is not defined', got", output)
		return
	}

	exit_code, err := outputs.OutputValueById(c, ni.Run_v1_Output_exit_code)
	if err != nil {
		t.Fatal(err)
	}

	if exit_code == 0 {
		t.Error("expected exit code to be non-zero, got", exit_code)
		return
	}
}

// Test the expected behaviour when no input is provided.
func Test_RunNode_NoInput1(t *testing.T) {

	_, nei, inputs, outputs := createRunNode(t)
	inputs.SetInputValue(ni.Run_v1_Input_shell, "python")
	// no script provided

	c := core.EmptyExecutionContext()

	err := nei.ExecuteImpl(c)
	if err != nil {
		t.Fatal(err)
	}

	output, err := outputs.OutputValueById(c, ni.Run_v1_Output_output)
	if err != nil {
		t.Fatal(err)
	}

	if output != "" {
		t.Error("expected output to be an empty string")
		return
	}

	exitCode, err := outputs.OutputValueById(c, ni.Run_v1_Output_exit_code)
	if err != nil {
		t.Fatal(err)
	}

	if exitCode != 0 {
		t.Fatal("expected exit code to be 0")
	}
}
