package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
)

//go:embed for@v1.yml
var forDefinition string

type ForNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *ForNode) ExecuteImpl(c core.ExecutionContext) error {
	firstIndex, err := core.InputValueById[int](c, n.Inputs, ni.For_v1_Input_first_index)
	if err != nil {
		return err
	}

	lastIndex, err := core.InputValueById[int](c, n.Inputs, ni.For_v1_Input_last_index)
	if err != nil {
		return err
	}

	if firstIndex > lastIndex {
		// zero executions
		return nil
	}

	body := n.Executions[ni.For_v1_Output_exec_body]
	if body != nil {

		for i := firstIndex; i <= lastIndex; i++ {

			err = n.Outputs.SetOutputValue(c, ni.For_v1_Output_index, i)
			if err != nil {
				return err
			}

			err = n.Execute(body, c)
			if err != nil {
				return u.Throw(err)
			}
		}
	}

	finish := n.Executions[ni.For_v1_Output_exec_finish]
	if finish != nil {
		err = n.Execute(finish, c)
		if err != nil {
			return u.Throw(err)
		}
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(forDefinition, func(context interface{}) (core.NodeRef, error) {
		return &ForNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
