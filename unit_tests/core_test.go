//go:build unit_tests

package unit_tests

import (
	"actionforge/graph-runner/core"
	"testing"

	// initialize all nodes
	_ "actionforge/graph-runner/nodes"
)

// Run a simple script/program and check that the output is correct.
func Test_NewNodeInstance(t *testing.T) {
	n, err := core.NewNodeInstance("run@v1")
	if err != nil {
		t.Fatal(err)
	}

	if n.GetNodeType() != "run@v1" {
		t.Errorf("Expected node name to be 'run@v1', got '%s'", n.GetNodeType())
	}
}
