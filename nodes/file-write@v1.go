package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
	"io"
	"os"
)

//go:embed file-write@v1.yml
var fileWriteDefinition string

type FileWriteNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
}

func (n *FileWriteNode) ExecuteImpl(c core.ExecutionContext) error {

	var fr io.Reader

	path, err := core.InputValueById[string](c, n.Inputs, ni.File_write_v1_Input_path)
	if err != nil {
		return err
	}

	content, err := core.InputValueById[any](c, n.Inputs, ni.File_write_v1_Input_content)
	if err != nil {
		return err
	}

	fw, err := os.Create(path)
	if err != nil {
		return err
	}

	cleanup := func() {
		_ = fw.Close()
		if f, ok := content.(*os.File); ok {
			_ = f.Close()
		}
	}

	defer cleanup()

	fr, err = utils.AnyToReader(content)
	if err != nil {
		return err
	}

	_, err = io.Copy(fw, fr)
	cleanup()

	if err == nil {
		err = n.Execute(n.GetTargetNode(ni.File_write_v1_Output_exec), c)
	} else {
		err = n.Execute(n.GetTargetNode(ni.File_write_v1_Output_exec_err), c)
	}

	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(fileWriteDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &FileWriteNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
