package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
	"runtime"
)

//go:embed parallel-multi-queue@v1.yml
var parallelMultiQueueDefinition string

type ParallelMultiQueueNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	pool *ThreadPool
}

func (n *ParallelMultiQueueNode) ExecuteImpl(ti core.ExecutionContext) error {

	workerCount, err := core.InputValueById[int](ti, n.Inputs, ni.Parallel_multi_queue_v1_Input_worker_count)
	if err != nil {
		return err
	}

	context, err := core.InputValueById[interface{}](ti, n.Inputs, ni.Parallel_multi_queue_v1_Input_context)
	if err != nil {
		return err
	}

	n.pool.AdjustWorkerCount(workerCount)

	var errors []error

	ti.Wg.Add(1)
	n.pool.AddTask(func() {
		defer ti.Wg.Done()
		nti := ti.PushNewExecutionContext()

		if context != nil {
			err := n.Outputs.SetOutputValue(nti, ni.Parallel_multi_queue_v1_Output_context, context)
			if err != nil {
				errors = append(errors, err)
				return
			}
		}

		err = n.Execute(n.Executions[ni.Parallel_multi_queue_v1_Output_exec_body], nti)
		if err != nil {
			errors = append(errors, err)
			return
		}
	})

	if len(errors) > 0 {
		// Combine all errors into a single error, or handle them as needed
		return fmt.Errorf("parallel execution errors: %v", errors)
	}

	return nil
}

func worker(taskQueue chan func()) {
	fmt.Println("Listening for tasks")
	for task := range taskQueue {
		fmt.Println("Received task", task)
		if task == nil {
			break
		}
		task()
	}
	fmt.Println("END!")
}

type ThreadPool struct {
	workerCount int
	taskQueue   chan func()
}

func NewThreadPool(initialWorkerCount int) *ThreadPool {
	tp := &ThreadPool{
		workerCount: initialWorkerCount,
		taskQueue:   make(chan func()),
	}

	for i := 0; i < tp.workerCount; i++ {
		go worker(tp.taskQueue)
	}

	return tp
}

func (tp *ThreadPool) AddTask(task func()) {
	tp.taskQueue <- task
}

func (tp *ThreadPool) AdjustWorkerCount(newWorkerCount int) {
	diff := newWorkerCount - tp.workerCount
	if diff > 0 {
		for i := 0; i < diff; i++ {
			go worker(tp.taskQueue)
		}
	} else if diff < 0 {
		for i := 0; i < -diff; i++ {
			tp.taskQueue <- nil
		}
	}
	tp.workerCount = newWorkerCount
}

func init() {
	err := core.RegisterNodeFactory(parallelMultiQueueDefinition, func(context interface{}) (core.NodeRef, error) {
		pool := NewThreadPool(runtime.NumCPU())
		return &ParallelMultiQueueNode{
			pool: pool,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
