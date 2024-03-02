package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	"context"
	_ "embed"
	"os"
)

//go:embed start@v1.yml
var startNodeDefinition string

type StartNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Outputs
}

func (n *StartNode) ExecuteEntry(inputValues map[core.OutputId]any) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := core.NewExecutionContext(ctx)

	err := n.Outputs.SetOutputValue(c, ni.Start_v1_Output_args, utils.GetSanitizedEnviron())
	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(c, ni.Start_v1_Output_env, os.Environ())
	if err != nil {
		return err
	}

	err = n.Execute(n, c)
	if err != nil {
		return err
	}
	return nil
}

func (n *StartNode) ExecuteImpl(c core.ExecutionContext) error {
	exec, ok := n.Executions[ni.Start_v1_Output_exec]
	if !ok {
		return nil
	}

	err := n.Execute(exec, c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(startNodeDefinition, func(ctx interface{}, nodeDef map[any]any) (core.NodeRef, error) {
		return &StartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
