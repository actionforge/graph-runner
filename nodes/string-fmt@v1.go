package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
	"strings"
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

	if strings.Contains(fmtString, "%") && len(input) > 0 {
		return fmt.Sprintf(fmtString, input...), nil
	} else {
		return fmtString, nil
	}
}

func init() {
	err := core.RegisterNodeFactory(stringFmtDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &StringFmt{}, nil
	})
	if err != nil {
		panic(err)
	}
}
