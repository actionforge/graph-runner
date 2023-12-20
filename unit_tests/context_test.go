//go:build unit_tests
// +build unit_tests

package unit_tests

import (
	"actionforge/graph-runner/core"
	"context"
	"testing"
)

func Test_EmptyExecutionContext(t *testing.T) {
	empty := core.EmptyExecutionContext()
	if len(empty.GetContextKeys(nil)) != 1 {
		t.Error("empty context must have a single key that's empty")
	}
}

func Test_ExecutionContext(t *testing.T) {
	root := core.NewExecutionContext(context.Background())

	if len(root.GetContextKeys(nil)) != 1 {
		t.Error("root context must have a single key")
	}

	sub1 := root.PushNewExecutionContext()
	if len(sub1.GetContextKeys(nil)) != 2 {
		t.Error("sub context must have two keys, the root context key, and a new context key")
	}

	if root.GetContextKeys(nil)[0] != sub1.GetContextKeys(nil)[1] {
		t.Error("root context keys must be identical")
	}

	sub2 := sub1.PushNewExecutionContext()
	if len(sub2.GetContextKeys(nil)) != 3 {
		t.Error("non-root context must have 3 keys, the root and sub context key, and a new context key")
	}

	if sub1.GetContextKeys(nil)[1] != sub2.GetContextKeys(nil)[2] {
		t.Error("root context keys must be identical")
	}

	if sub1.GetContextKeys(nil)[0] != sub2.GetContextKeys(nil)[1] {
		t.Error("non-root context keys must be identical")
	}

	if sub1.GetLastContextKey() == sub2.GetLastContextKey() {
		t.Error("non-root context keys must be different")
	}
}
