//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"os"
)

//go:embed env-get@v1.yml
var envGetDefinition string

type EnvGetNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *EnvGetNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	env, err := core.InputValueById[string](c, n.Inputs, ni.Env_get_v1_Input_env)
	if err != nil {
		return nil, err
	}

	env, _ = os.LookupEnv(env)
	// ignore error, returning empty string

	return env, nil
}

func init() {
	err := core.RegisterNodeFactory(envGetDefinition, func(context interface{}) (core.NodeRef, error) {
		return &EnvGetNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
