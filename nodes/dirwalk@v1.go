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

func (n *WalkNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {

	glob, err := core.InputValueById[string](c, n.Inputs, ni.Dirwalk_v1_Input_glob)
	if err != nil {
		return err
	}

	dir, err := core.InputValueById[string](c, n.Inputs, ni.Dirwalk_v1_Input_dir)
	if err != nil {
		return err
	}

	pattern := strings.Split(glob, ";")

	items := make(map[string]struct{})
	err = walk(dir, pattern, items)
	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(c, "items", maps.Keys(items))
	if err != nil {
		return err
	}

	err = n.Execute(ni.Dirwalk_v1_Output_exec, c)
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

		include := false

		for _, p := range pattern {
			matched, err := filepath.Match(p, filepath.Base(path))
			if err != nil {
				return err
			}
			include = include || matched
			if matched {
				break
			}
		}

		if include {
			items[path] = struct{}{}
		}

		return nil
	})
}

func init() {
	err := core.RegisterNodeFactory(walkDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &WalkNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
