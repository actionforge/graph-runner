package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
)

//go:embed const-string@v1.yml
var constStringDefinition string

//go:embed const-number@v1.yml
var constNumberDefinition string

//go:embed const-bool@v1.yml
var constBoolDefinition string

type ConstNode[T any] struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs

	portValue core.InputId
}

func (n *ConstNode[T]) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	inputs, err := core.InputValueById[T](c, n.Inputs, n.portValue)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func init() {
	err := core.RegisterNodeFactory(constStringDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &ConstNode[string]{
			portValue: ni.Const_string_v1_Input_input,
		}, nil
	})
	if err != nil {
		panic(err)
	}

	err = core.RegisterNodeFactory(constNumberDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &ConstNode[int64]{
			portValue: ni.Const_number_v1_Input_input,
		}, nil
	})
	if err != nil {
		panic(err)
	}

	err = core.RegisterNodeFactory(constBoolDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &ConstNode[bool]{
			portValue: ni.Const_bool_v1_Input_input,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
