package nodes

import (
	"actionforge/graph-runner/core"
	_ "embed"
)

//go:embed group-start@v1.yml
var groupStartDefinition string

type GroupStartNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *GroupStartNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *GroupStartNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	err := n.Execute(core.OutputId(inputId), c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(groupStartDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &GroupStartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
