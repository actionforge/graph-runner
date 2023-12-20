package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed string-match@v1.yml
var stringMatchDefinition string

type StringMatchNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *StringMatchNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	str1, err := core.InputValueById[string](c, n.Inputs, ni.String_match_v1_Input_str1)
	if err != nil {
		return nil, err
	}

	str2, err := core.InputValueById[string](c, n.Inputs, ni.String_match_v1_Input_str2)
	if err != nil {
		return nil, err
	}

	op, err := core.InputValueById[string](c, n.Inputs, ni.String_match_v1_Input_op)
	if err != nil {
		return nil, err
	}

	switch op {
	case "contains":
		return strings.Contains(str1, str2), nil
	case "notcontains":
		return !strings.Contains(str1, str2), nil
	case "startswith":
		return strings.HasPrefix(str1, str2), nil
	case "endswith":
		return strings.HasSuffix(str1, str2), nil
	case "equals":
		return str1 == str2, nil
	case "regex":
		return regexp.MatchString(str2, str1)
	default:
		return nil, fmt.Errorf("unknown operation: %v", op)
	}
}

func init() {
	err := core.RegisterNodeFactory(stringMatchDefinition, func(context interface{}) (core.NodeRef, error) {
		return &StringMatchNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
