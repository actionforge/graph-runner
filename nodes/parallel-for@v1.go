package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"sync"
)

//go:embed parallel-for@v1.yml
var parallelForDefinition string

type ParallelForNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *ParallelForNode) ExecuteImpl(ti core.ExecutionContext) error {
	firstIndex, err := core.InputValueById[int](ti, n.Inputs, ni.Parallel_for_v1_Input_first_index)
	if err != nil {
		return err
	}

	lastIndex, err := core.InputValueById[int](ti, n.Inputs, ni.Parallel_for_v1_Input_last_index)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	if firstIndex > lastIndex {
		// zero executions
		return nil
	}

	// TODO: (Seb) Turn into queue, not all at once

	wg := sync.WaitGroup{}

	var mutex sync.Mutex
	var errors []error

	body := n.Executions[ni.Parallel_for_v1_Output_exec_body]
	if body != nil {

		for i := firstIndex; i <= lastIndex; i++ {

			nti := ti.PushNewExecutionContext()
			err = n.Outputs.SetOutputValue(nti, ni.For_v1_Output_index, i, core.SetOutputValueOpts{})
			if err != nil {
				return err
			}

			wg.Add(1)
			go func() {
				defer wg.Done()

				err = n.Execute(body, nti)
				if err != nil {
					mutex.Lock()
					errors = append(errors, err)
					mutex.Unlock()
					return
				}
			}()
		}
	}

	wg.Wait()

	if len(errors) > 0 {
		// Combine all errors into a single error, or handle them as needed
		return fmt.Errorf("parallel execution errors: %v", errors)
	}

	finish := n.Executions[ni.Parallel_for_v1_Output_exec_finish]
	if finish != nil {
		err = n.Execute(finish, ti)
		if err != nil {
			return u.Throw(err)
		}
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(parallelForDefinition, func(context interface{}) (core.NodeRef, error) {
		return &ParallelForNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
