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

func isZeroValue(input any) bool {
	switch v := input.(type) {
	case bool:
		return !v
	case string:
		return v == ""
	case int:
		return v == 0
	case int8:
		return v == 0
	case int16:
		return v == 0
	case int32:
		return v == 0
	case int64:
		return v == 0
	case uint:
		return v == 0
	case uint8:
		return v == 0
	case uint16:
		return v == 0
	case uint32:
		return v == 0
	case uint64:
		return v == 0
	case float32:
		return v == float32(0.0)
	case float64:
		return v == float64(0.0)
	default:
		return false
	}
}

func (n *NegateNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	input, errA := core.InputValueById[any](c, n.Inputs, ni.Bool_or_v1_Input_input)
	if errA != nil {
		return nil, errA
	}

	return !isZeroValue(input), nil
}

func init() {
	err := core.RegisterNodeFactory(negateDefinition, func(context interface{}) (core.NodeRef, error) {
		return &NegateNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
