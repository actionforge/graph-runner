package nodes

import (
	"actionforge/graph-runner/core"
	_ "embed"
)

//go:embed group-output@v1.yml
var groupOutputDefinition string

type GroupOutputNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *GroupOutputNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {
	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *GroupOutputNode) ExecuteImpl(c core.ExecutionContext) error {
	err := n.Execute(n.GetExecutionPort("exec"), c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(groupOutputDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &GroupOutputNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
