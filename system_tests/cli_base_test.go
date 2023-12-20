//go:build system_tests
// +build system_tests

package system_tests

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

func TestMain(m *testing.M) {
	cmd := exec.Command("go", "build", "--tags=github_impl", ".")
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func Test_Cli(t *testing.T) {
	cmd := exec.Command("./graph-runner", "run", "--graph_file", "system_tests/test_simple.yml")
	cmd.Dir = utils.FindProjectRoot()
	output, err := cmd.CombinedOutput()
	fmt.Println(string(output))

	if err != nil {
		t.Fatal(err)
	}
}
