package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"errors"
	"path/filepath"
)

//go:embed filepath-op@v1.yml
var filepathOpDefinition string

type FilepathOp struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *FilepathOp) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	path, err := core.InputValueById[string](c, n.Inputs, ni.Filepath_op_v1_Input_path)
	if err != nil {
		return nil, err
	}

	op, err := core.InputValueById[string](c, n.Inputs, ni.Filepath_op_v1_Input_op)
	if err != nil {
		return nil, err
	}

	switch op {
	case "base":
		return filepath.Base(path), nil
	case "clean":
		return filepath.Clean(path), nil
	case "dir":
		return filepath.Dir(path), nil
	case "ext":
		return filepath.Ext(path), nil
	case "from_slash":
		return filepath.FromSlash(path), nil
	case "to_slash":
		return filepath.ToSlash(path), nil
	case "volume":
		return filepath.VolumeName(path), nil
	}

	return nil, errors.New("unknown op")
}

func init() {
	err := core.RegisterNodeFactory(filepathOpDefinition, func(context interface{}) (core.NodeRef, error) {
		return &FilepathOp{}, nil
	})
	if err != nil {
		panic(err)
	}
}
