package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
	"reflect"
	"runtime"
	"sync"
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

func (n *ParallelMultiQueueNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	workerCount, err := core.InputValueById[int](c, n.Inputs, ni.Parallel_multi_queue_v1_Input_worker_count)
	if err != nil {
		return err
	}

	context, err := core.InputValueById[any](c, n.Inputs, ni.Parallel_multi_queue_v1_Input_context)
	if err != nil {
		return err
	}

	n.pool.AdjustWorkerCount(workerCount)

	var mutex sync.Mutex
	var errors []error

	wg := sync.WaitGroup{}

	s := reflect.ValueOf(context)
	for i := 0; i < s.Len(); i++ {
		nti := c.PushNewExecutionContext()
		err = n.Outputs.SetOutputValue(nti, ni.Parallel_multi_queue_v1_Output_context, s.Index(i).Interface())
		if err != nil {
			return err
		}

		wg.Add(1)
		c.Wg.Add(1)
		n.pool.AddTask(func() {
			defer wg.Done()
			defer c.Wg.Done()
			err := n.Execute(ni.Parallel_multi_queue_v1_Output_exec_body, nti)
			if err != nil {
				mutex.Lock()
				errors = append(errors, err)
				mutex.Unlock()
			}
		})
	}

	if len(errors) > 0 {
		// Combine all errors into a single error, or handle them as needed
		return fmt.Errorf("parallel execution errors: %v", errors)
	}

	wg.Wait()

	err = n.Execute(ni.Parallel_multi_queue_v1_Output_exec_finish, c)
	if err != nil {
		return err
	}

	return nil
}

func worker(taskQueue chan func()) {
	for task := range taskQueue {
		if task == nil {
			break
		}
		task()
	}
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
	err := core.RegisterNodeFactory(parallelMultiQueueDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		pool := NewThreadPool(runtime.NumCPU())
		return &ParallelMultiQueueNode{
			pool: pool,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
