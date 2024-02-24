package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/maps"
)

//go:embed dirwalk@v1.yml
var walkDefinition string

type WalkNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *WalkNode) ExecuteImpl(ti core.ExecutionContext) error {

	glob, err := core.InputValueById[string](ti, n.Inputs, "glob")
	if err != nil {
		return err
	}

	dir, err := core.InputValueById[string](ti, n.Inputs, "dir")
	if err != nil {
		return err
	}

	pattern := strings.Split(glob, ";")
	pattern = append(pattern, ".DS_Store")

	items := make(map[string]struct{})
	err = walk(dir, pattern, items)
	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(ti, "items", maps.Keys(items))
	if err != nil {
		return err
	}

	err = n.Execute(n.Executions[ni.Dirwalk_v1_Output_exec], ti)
	if err != nil {
		return err
	}

	return nil
}

func walk(root string, pattern []string, items map[string]struct{}) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		matchIgnore := false

		for _, p := range pattern {
			matched, err := filepath.Match("!"+p, filepath.Base(path))
			if err != nil {
				return err
			}

			matchIgnore = matchIgnore || matched
		}

		if matchIgnore {
			return nil
		}

		items[path] = struct{}{}

		return nil
	})
}

func init() {
	err := core.RegisterNodeFactory(walkDefinition, func(context interface{}) (core.NodeRef, error) {
		return &WalkNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
