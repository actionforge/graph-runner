package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
)

//go:embed bool-or@v1.yml
var boolOrDefinition string

//go:embed bool-and@v1.yml
var boolAndDefinition string

//go:embed bool-xor@v1.yml
var boolXorDefinition string

//go:embed bool-xand@v1.yml
var boolXandDefinition string

type BoolNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs

	op    func(bool, bool) bool
	opStr string // just for debugging
}

func (n *BoolNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	inputs, err := core.InputGroupValue[bool](c, n.Inputs, ni.Bool_or_v1_Input_input)
	if err != nil {
		return nil, err
	}

	var result bool

	if len(inputs) > 0 {
		result = inputs[0]

		for _, input := range inputs[1:] {
			result = n.op(result, input)
		}
	}

	return result, nil
}

func init() {
	// OR
	err := core.RegisterNodeFactory(boolOrDefinition, func(context interface{}) (core.NodeRef, error) {
		return &BoolNode{
			op: func(a bool, b bool) bool {
				return a || b
			},
			opStr: "OR",
		}, nil
	})
	if err != nil {
		panic(err)
	}

	// AND
	err = core.RegisterNodeFactory(boolAndDefinition, func(context interface{}) (core.NodeRef, error) {
		return &BoolNode{
			op: func(a bool, b bool) bool {
				return a && b
			},
			opStr: "AND",
		}, nil
	})
	if err != nil {
		panic(err)
	}

	// XOR
	err = core.RegisterNodeFactory(boolXorDefinition, func(context interface{}) (core.NodeRef, error) {
		return &BoolNode{
			op: func(a bool, b bool) bool {
				return a != b
			},
			opStr: "XOR",
		}, nil
	})
	if err != nil {
		panic(err)
	}

	// XAND
	err = core.RegisterNodeFactory(boolXandDefinition, func(context interface{}) (core.NodeRef, error) {
		return &BoolNode{
			op: func(a bool, b bool) bool {
				return a == b
			},
			opStr: "XAND",
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
