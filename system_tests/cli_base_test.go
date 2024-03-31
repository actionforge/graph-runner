//go:build system_tests

package system_tests

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	cmd := exec.Command("go",
		"build",
		"--tags=github_impl",
		"-ldflags",
		"-X actionforge/graph-runner/core.Production=true",
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

func Test_Freeze(t *testing.T) {
	// Exclude test during local execution.
	// The freeze command depends on the current
	// commit being accessible as a zip archive on
	// GitHub, which may not be the case for commits
	// made locally but not yet pushed.
	if os.Getenv("FREEZE_TEST") == "" {
		t.Skip("Skipping test locally")
	}

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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Dir = utils.FindProjectRoot()

	err := cmd.Run()
	if err != nil {
		return err
	}

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
