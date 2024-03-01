package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
)

//go:embed env-array@v1.yml
var envArrayDefinition string

type EnvArrayNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *EnvArrayNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	envs, err := core.InputGroupValue[string](c, n.Inputs, ni.Env_array_v1_Input_env)
	if err != nil {
		return nil, err
	}

	for i, env := range envs {
		envs[i] = ReplaceContextVariables(env)
	}

	return envs, nil
}

func init() {
	err := core.RegisterNodeFactory(envArrayDefinition, func(context interface{}) (core.NodeRef, error) {
		return &EnvArrayNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
