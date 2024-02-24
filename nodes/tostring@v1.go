package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
)

//go:embed tostring@v1.yml
var toStringDefinition string

type ToStringNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *ToStringNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	value, err := core.InputValueById[string](c, n.Inputs, ni.Tostring_v1_Input_value)
	if err != nil {
		return nil, err
	}

	str := utils.AnyToString(value)
	err = n.Outputs.SetOutputValue(c, ni.Tostring_v1_Output_string, str)
	if err != nil {
		return nil, err
	}

	return str, nil
}

func init() {
	err := core.RegisterNodeFactory(toStringDefinition, func(context interface{}) (core.NodeRef, error) {
		return &ToStringNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
