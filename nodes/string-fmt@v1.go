package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
)

//go:embed string-fmt@v1.yml
var stringFmtDefinition string

type StringFmt struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *StringFmt) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	fmtString, err := core.InputValueById[string](c, n.Inputs, ni.String_fmt_v1_Input_fmt)
	if err != nil {
		return nil, err
	}

	input, err := core.InputGroupValue[any](c, n.Inputs, ni.String_fmt_v1_Input_input)
	if err != nil {
		return nil, err
	}
	return fmt.Sprintf(fmtString, input...), nil
}

func init() {
	err := core.RegisterNodeFactory(stringFmtDefinition, func(context interface{}) (core.NodeRef, error) {
		return &StringFmt{}, nil
	})
	if err != nil {
		panic(err)
	}
}
