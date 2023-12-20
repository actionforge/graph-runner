package core

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

type contextKey string

// ExecutionContext is a structure whose main purpose is to provide the correct output values requested
// by nodes that were executed in subsequent goroutines.
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
	contextKeys []contextKey
}

func EmptyExecutionContext() ExecutionContext {
	c := ExecutionContext{Context: context.Background(),
		contextKeys: []contextKey{""},
	}
	return c
}

func NewExecutionContext(ctx context.Context) ExecutionContext {
	c := ExecutionContext{Context: ctx,
		contextKeys: []contextKey{""},
	}
	return c
}

// PushNewExecutionContext creates a new execution context and pushes it to the stack.
// Should be used right before a new goroutine is created and called.
//
//		newEc := ti.PushNewExecutionContext()
//		err = n.Outputs.SetOutputValue(newEc, [output-id], [output-value])
//		if err != nil {
//		    return err
//		}
//		wg.Add(1)
//		go func() {
//	    	err := n.ExecBody.Execute(newEc)
//	        ...
//		}();
func (c *ExecutionContext) PushNewExecutionContext() ExecutionContext {
	ti := uuid.Must(uuid.NewRandom()).String()
	ck := contextKey(ti)
	threadIds := append(c.contextKeys, ck)
	return ExecutionContext{
		Context:     context.WithValue(c.Context, ck, "random-value"),
		contextKeys: threadIds,
	}
}

// GetLastContextKey returns the last context key for the current goroutine.
func (c *ExecutionContext) GetLastContextKey() contextKey {
	if len(c.contextKeys) == 0 {
		return ""
	}
	return c.contextKeys[len(c.contextKeys)-1]
}

// GetContextKeys returns all context keys for the current goroutine.
// The first key is the most recent one, the last key is the root context key.
// @param recentFirst: If true, the order is reversed.
func (c *ExecutionContext) GetContextKeys(recentLast *bool) []contextKey {
	k := slices.Clone(c.contextKeys)
	if recentLast == nil || !*recentLast {
		slices.Reverse(k)
	}
	return k
}
