package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
)

//go:embed branch@v1.yml
var ifDefinition string

type BranchNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *BranchNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	condition, err := core.InputValueById[bool](c, n.Inputs, ni.Branch_v1_Input_condition)
	if err != nil {
		return err
	}

	if condition {
		err = n.Execute(ni.Branch_v1_Output_exec_then, c)
		if err != nil {
			return u.Throw(err)
		}
	} else {
		err = n.Execute(ni.Branch_v1_Output_exec_otherwise, c)
		if err != nil {
			return u.Throw(err)
		}
	}
	return nil
}

func init() {
	err := core.RegisterNodeFactory(ifDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &BranchNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
