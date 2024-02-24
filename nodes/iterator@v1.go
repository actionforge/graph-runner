package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
)

//go:embed iterator@v1.yml
var iteratorDefinition string

type IteratorNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *IteratorNode) ExecuteImpl(ti core.ExecutionContext) error {

	array, err := core.InputValueById[interface{}](ti, n.Inputs, "array")
	if err != nil {
		return err
	}

	for i, item := range array.([]string) {

		err = n.Outputs.SetOutputValue(ti, ni.Iterator_v1_Output_key, i)
		if err != nil {
			return err
		}

		err = n.Outputs.SetOutputValue(ti, ni.Iterator_v1_Output_value, item)
		if err != nil {
			return err
		}

		err = n.Execute(n.Executions[ni.Iterator_v1_Output_exec], ti)
		if err != nil {
			return u.Throw(err)
		}
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(iteratorDefinition, func(context interface{}) (core.NodeRef, error) {
		return &IteratorNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
