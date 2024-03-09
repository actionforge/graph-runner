package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"path/filepath"
)

//go:embed filepath-join@v1.yml
var filepathJoinDefinition string

type FilepathJoin struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *FilepathJoin) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	paths, err := core.InputGroupValue[string](c, n.Inputs, ni.Filepath_join_v1_Input_paths)
	if err != nil {
		return nil, err
	}

	return filepath.Join(paths...), nil
}

func init() {
	err := core.RegisterNodeFactory(filepathJoinDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &FilepathJoin{}, nil
	})
	if err != nil {
		panic(err)
	}
}
