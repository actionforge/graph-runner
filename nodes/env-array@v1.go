package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed env-array@v1.yml
var envArrayDefinition string

type EnvArrayNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *EnvArrayNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (any, error) {
	envs, err := core.InputGroupValue[string](c, n.Inputs, ni.Env_array_v1_Input_env)
	if err != nil {
		return nil, err
	}

	contextEnvironMap := c.GetContextEnvironMapCopy()
	for _, env := range envs {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			contextEnvironMap[kv[0]] = utils.ReplaceContextVariables(kv[1])
		}
	}

	return func() any {
		envArray := []string{}
		for k, v := range contextEnvironMap {
			envArray = append(envArray, fmt.Sprintf("%s=%s", k, v))
		}
		return envArray
	}(), nil
}

func init() {
	err := core.RegisterNodeFactory(envArrayDefinition, func(context interface{}) (core.NodeRef, error) {
		return &EnvArrayNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
