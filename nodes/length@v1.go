package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"reflect"
)

//go:embed length@v1.yml
var lengthDefinition string

type LengthNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *LengthNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	inputs, err := core.InputValueById[any](c, n.Inputs, ni.Length_v1_Input_input)
	if err != nil {
		return nil, err
	}

	v := reflect.ValueOf(inputs)

	return v.Len(), nil
}

func init() {
	err := core.RegisterNodeFactory(lengthDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &LengthNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
