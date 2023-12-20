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

func (n *BranchNode) ExecuteImpl(c core.ExecutionContext) error {
	condition, err := core.InputValueById[bool](c, n.Inputs, ni.Branch_v1_Input_condition)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	if condition {
		err = n.Execute(n.Executions[ni.Branch_v1_Output_then], c)
		if err != nil {
			return u.Throw(err)
		}
	} else {
		err = n.Execute(n.Executions[ni.Branch_v1_Output_otherwise], c)
		if err != nil {
			return u.Throw(err)
		}
	}
	return nil
}

func init() {
	err := core.RegisterNodeFactory(ifDefinition, func(context interface{}) (core.NodeRef, error) {
		return &BranchNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
