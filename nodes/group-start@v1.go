package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"context"
	_ "embed"
)

//go:embed group-start@v1.yml
var subGraphStartDefinition string

type SubGraphStartNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *SubGraphStartNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *SubGraphStartNode) ExecuteEntry(outputValues map[core.OutputId]any) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := core.NewExecutionContext(ctx)

	for k, v := range outputValues {
		err := n.Outputs.SetOutputValue(c, k, v)
		if err != nil {
			return err
		}
	}

	err := n.Execute(n, c)
	if err != nil {
		return err
	}
	return nil
}

func (n *SubGraphStartNode) ExecuteImpl(c core.ExecutionContext) error {
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
	err := core.RegisterNodeFactory(subGraphStartDefinition, func(ctx interface{}, nodeDef map[any]any) (core.NodeRef, error) {
		return &SubGraphStartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
