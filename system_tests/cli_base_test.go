//go:build system_tests

package system_tests

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	cmd := exec.Command("go",
		"build",
		"--tags=github_impl",
		".",
	)
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func Test_Cli(t *testing.T) {
	cmd := exec.Command("./graph-runner", "system_tests/test_simple.yml")
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))

	if err != nil {
		t.Fatal(err)
	}
}

/*
func Test_Frozen(t *testing.T) {
	actionHomeDir := utils.GetActionforgeDir()

	err := os.RemoveAll(actionHomeDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Building frozen binary (1st run)")
	err = buildFrozen("system_tests/test_freeze.yml")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Testing frozen binary (1st run)")
	err = testFrozen("Hello World!")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Building frozen binary (2nd run)")
	err = buildFrozen("system_tests/test_freeze2.yml")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Testing frozen binary (2nd run)")
	err = testFrozen("Foo Bar Bas")
	if err != nil {
		t.Fatal(err)
	}
}

func buildFrozen(graphPath string) error {
	cmd := exec.Command(
		"./graph-runner",
		"freeze",
		graphPath,
		"--output",
		"frozen",
	)
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))

	return err
}

func testFrozen(expected string) error {
	cmd := exec.Command("./frozen")
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err == nil {
		if !strings.Contains(string(output), expected) {
			err = fmt.Errorf("Unexpected output: %s", string(output))
		}
	}

	return err
}
*/
