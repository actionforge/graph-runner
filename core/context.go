package core

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"
)

type contextStackItem struct {
	Id  string
	Env map[string]string
}

type contextKey string

// ExecutionContext is a structure whose main purpose is to provide the correct output values
// and environment variables requested by nodes that were executed in subsequent goroutines.
//
// Basically, it's a linear sequence of ids, each representing a goroutine in the execution stack
//
//							/-> AB ---> AB'		<--- An execution context
//	    ---> A ---> A ---> A'
//							\-> AC ---> AC'	    <--- Another execution context
//
// Each item in the graph above represents a node and the corresponding context keys. The key (A) represents
// the main routine from which the execution began. After A' each goroutine gets its own context with an additional
// id. Now AB can fetch data from AB and A, whereas AC' might fetch a different value from A'.
//
// An example is the 'Parallel For' node where each iteration runs in a separate goroutine.
// Nodes within these new executions can fetch their respective iteration index they are associated with.
// Without this approach, all nodes in subsequent goroutines would fetch the same value, which is the last.
type ExecutionContext struct {
	context.Context
	contextStack []contextStackItem
}

func EmptyExecutionContext() ExecutionContext {
	c := ExecutionContext{Context: context.Background(),
		contextStack: []contextStackItem{{
			Id:  "",
			Env: make(map[string]string),
		}},
	}
	return c
}

func NewExecutionContext(ctx context.Context, env map[string]string) ExecutionContext {
	c := ExecutionContext{Context: ctx,
		contextStack: []contextStackItem{{
			Id:  uuid.New().String(),
			Env: env,
		}},
	}
	return c
}

// PushNewExecutionContext creates a new execution context and pushes it to the stack.
// Should be used right before a new goroutine is created and called.
//
//		newEc := ti.PushNewExecutionContext()
//		err = n.Outputs.SetOutputValue(newEc, <output-id>, <output-value>)
//		if err != nil {
//		    return err
//		}
//		wg.Add(1)
//		go func() {
//	    	err := n.ExecBody.Execute(newEc)
//	        ...
//		}();
func (c *ExecutionContext) PushNewExecutionContext() ExecutionContext {

	contextEnv := make(map[string]string)

	if len(c.contextStack) > 0 {
		for k, v := range c.contextStack[len(c.contextStack)-1].Env {
			contextEnv[k] = v
		}
	}

	ck := append(c.contextStack, contextStackItem{
		Id:  uuid.New().String(),
		Env: contextEnv,
	})
	return ExecutionContext{
		Context:      context.WithValue(c.Context, contextKey("contextData"), ck),
		contextStack: ck,
	}
}

// GetLastContextKey returns the last context key for the current goroutine.
func (c *ExecutionContext) GetLastContextKey() contextStackItem {
	if len(c.contextStack) == 0 {
		return contextStackItem{}
	}
	return c.contextStack[len(c.contextStack)-1]
}

// GetContextKeysCopy returns all context keys for the current goroutine.
// The first key is the most recent one, the last key is the root context key.
// @param recentFirst: If true, the order is reversed.
func (c *ExecutionContext) GetContextKeysCopy(recentLast *bool) []contextStackItem {
	k := slices.Clone(c.contextStack)
	if recentLast == nil || !*recentLast {
		slices.Reverse(k)
	}
	return k
}

// GetContextEnvironMapCopy returns the environment variables for the current and subsequent goroutines.
func (c *ExecutionContext) GetContextEnvironMapCopy() map[string]string {
	return maps.Clone(c.GetLastContextKey().Env)
}

// SetContextEnvironMap sets the environment variables for the current and subsequent goroutines.
func (c *ExecutionContext) SetContextEnvironMap(env map[string]string) {
	if len(c.contextStack) > 0 {
		c.contextStack[len(c.contextStack)-1].Env = env
	}
}
