package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
)

//go:embed negate@v1.yml
var negateDefinition string

type NegateNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *NegateNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	input, errA := core.InputValueById[any](c, n.Inputs, ni.Bool_or_v1_Input_input)
	if errA != nil {
		return nil, errA
	}

	ai := bool(input == true)
	bi := bool(input != "")
	ci := bool(input != nil)
	di := bool(input != 0)
	ei := bool(input != 0.0)

	return !(ai || bi || ci || di || ei), nil
}

func init() {
	err := core.RegisterNodeFactory(negateDefinition, func(context interface{}) (core.NodeRef, error) {
		return &NegateNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
