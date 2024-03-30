//go:build system_tests

package system_tests

import (
	"actionforge/graph-runner/cmd"
	"actionforge/graph-runner/utils"
	"fmt"
	"path/filepath"

	"github.com/sergi/go-diff/diffmatchpatch"

	// initialize all nodes
	_ "actionforge/graph-runner/nodes"
)

func RunGraphFile(graphFileName string) error {
	root := utils.FindProjectRoot()
	graphFile := filepath.Join(root, graphFileName)

	err := cmd.ExecuteRun(graphFile)
	if err != nil {
		return err
	}

	return nil
}

func DiffStrings(actual string, expected string) bool {
	dmp := diffmatchpatch.New()

	actual = utils.NormalizeLineEndings(actual)
	expected = utils.NormalizeLineEndings(expected)

	diffs := dmp.DiffMain(actual, expected, false)

	different := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			different = true
			break
		}
	}
	if different {
		fmt.Println("\n\n-----------\nExpected output vs actual output (inline-diff):")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
	return !different
}

func init() {
	utils.EnableStringLogging(true)
}
