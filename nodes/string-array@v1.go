//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
)

//go:embed string-array@v1.yml
var stringArrayDefinition string

type StringArrayNode struct {
	core.NodeBaseComponent
	core.Inputs
}

func (n *StringArrayNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	envs, err := core.InputGroupValue[string](c, n.Inputs, ni.String_array_v1_Input_env)
	if err != nil {
		return nil, err
	}

	return envs, nil
}

func init() {
	err := core.RegisterNodeFactory(stringArrayDefinition, func(context interface{}) (core.NodeRef, error) {
		return &StringArrayNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
