package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"sync"
)

//go:embed parallel-exec@v1.yml
var parallelExecDefinition string

type ParallelExecNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *ParallelExecNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	wg := sync.WaitGroup{}

	var mutex sync.Mutex
	var errors []error

	for outputId, t := range n.GetExecutionTargets() {
		if t.Target == nil {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			nti := c.PushNewExecutionContext()
			err := n.Execute(outputId, nti)
			if err != nil {
				mutex.Lock()
				errors = append(errors, err)
				mutex.Unlock()
				return
			}
		}()
	}

	wg.Wait()

	if len(errors) > 0 {
		// Combine all errors into a single error, or handle them as needed
		return fmt.Errorf("parallel execution errors: %v", errors)
	}

	err := n.Execute(ni.Parallel_for_v1_Output_exec_finish, c)
	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(parallelExecDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &ParallelExecNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
