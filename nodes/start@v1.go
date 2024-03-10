package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"context"
	_ "embed"
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

	err := n.Execute(n, c)
	if err != nil {
		return err
	}

	c.Wg.Wait()
	return nil
}

func (n *StartNode) ExecuteImpl(c core.ExecutionContext) error {
	err := n.Execute(n.GetTargetNode(ni.Start_v1_Output_exec), c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(startNodeDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &StartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
