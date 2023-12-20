package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"sync"
)

//go:embed wait-for@v1.yml
var waitForDefinition string

type WaitForNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs

	Lock sync.Mutex
	core.Executions

	CurrentCounter int
}

func (n *WaitForNode) ExecuteImpl(c core.ExecutionContext) error {

	n.Lock.Lock()

	executeAfter, err := core.InputValueById[int](c, n.Inputs, ni.Wait_for_v1_Input_after)
	if err != nil {
		n.Lock.Unlock()
		return err
	}

	if n.CurrentCounter == -1 {
		n.CurrentCounter = executeAfter
	} else {
		loop, err := core.InputValueById[bool](c, n.Inputs, ni.Wait_for_v1_Input_loop)
		if err != nil {
			n.Lock.Unlock()
			return err
		}

		if loop {
			n.CurrentCounter = executeAfter
		} else {
			if n.CurrentCounter == 0 {
				n.Lock.Unlock()
				return nil
			}
		}
	}

	n.CurrentCounter--

	if n.CurrentCounter > 0 {
		n.Lock.Unlock()
		return nil
	}

	n.Lock.Unlock()

	err = n.Execute(n.Executions[ni.Wait_for_v1_Output_exec], c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(waitForDefinition, func(context interface{}) (core.NodeRef, error) {
		return &WaitForNode{
			Lock:           sync.Mutex{},
			CurrentCounter: -1,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
