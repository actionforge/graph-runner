package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"os"
)

//go:embed file-read@v1.yml
var fileReadStreamDefinition string

type ReadFileNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *ReadFileNode) ExecuteImpl(c core.ExecutionContext) error {
	path, err := core.InputValueById[string](c, n.Inputs, ni.File_read_v1_Input_path)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(c, ni.File_read_v1_Output_stream, file)
	if err != nil {
		return err
	}

	err = n.Execute(n.Executions[ni.File_read_v1_Output_exec], c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(fileReadStreamDefinition, func(context interface{}) (core.NodeRef, error) {
		return &ReadFileNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
