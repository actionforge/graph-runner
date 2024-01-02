//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"context"
	_ "embed"
	"fmt"
)

//go:embed start@v1.yml
var startNodeDefinition string

type StartNode struct {
	core.NodeBaseComponent
	core.Executions
}

func (n *StartNode) ExecuteEntry() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := core.NewExecutionContext(ctx)
	return n.Execute(n, c)
}

func (n *StartNode) ExecuteImpl(c core.ExecutionContext) error {
	exec, ok := n.Executions[ni.Start_v1_Output_exec]
	if !ok {
		return fmt.Errorf("missing execution node")
	}

	err := n.Execute(exec, c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(startNodeDefinition, func(ctx interface{}) (core.NodeRef, error) {
		return &StartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
