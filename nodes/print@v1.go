package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
)

//go:embed print@v1.yml
var printDefinition string

type PrintNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
}

func (n *PrintNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {

	value, err := core.InputValueById[any](c, n.Inputs, ni.Print_v1_Input_value)
	if err != nil {
		return err
	}

	fmt.Printf("%v\n", value)

	err = n.Execute(ni.Print_v1_Output_exec, c)
	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(printDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &PrintNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
