package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
)

//go:embed cast-string@v1.yml
var castStringDefinition string

//go:embed cast-bool@v1.yml
var castBoolDefinition string

type CastNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs

	Cast func(value any) any
}

func (n *CastNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	value, err := core.InputValueById[string](c, n.Inputs, ni.Cast_string_v1_Input_value)
	if err != nil {
		return nil, err
	}

	str := n.Cast(value)
	err = n.Outputs.SetOutputValue(c, ni.Cast_string_v1_Output_output, str)
	if err != nil {
		return nil, err
	}

	return str, nil
}

func init() {
	err := core.RegisterNodeFactory(castStringDefinition, func(context interface{}) (core.NodeRef, error) {
		return &CastNode{
			Cast: func(value any) any {
				return utils.AnyToString(value)
			},
		}, nil
	})
	if err != nil {
		panic(err)
	}

	err = core.RegisterNodeFactory(castBoolDefinition, func(context interface{}) (core.NodeRef, error) {
		return &CastNode{
			Cast: func(value any) any {
				return utils.AnyToBool(value)
			},
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
