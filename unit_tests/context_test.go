//go:build unit_tests

package unit_tests

import (
	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	"context"
	"testing"

	_ "actionforge/graph-runner/nodes"
)

func Test_EmptyExecutionContext(t *testing.T) {
	empty := core.EmptyExecutionContext()
	if len(empty.GetContextKeysCopy(nil)) != 1 {
		t.Error("empty context must have a single key that's empty")
	}
}

func Test_ExecutionContext(t *testing.T) {
	root := core.NewExecutionContext(context.Background(), utils.GetSanitizedEnvironMap())

	if len(root.GetContextKeysCopy(nil)) != 1 {
		t.Error("root context must have a single key")
	}

	sub1 := root.PushNewExecutionContext()
	if len(sub1.GetContextKeysCopy(nil)) != 2 {
		t.Error("sub context must have two keys, the root context key, and a new context key")
	}

	if root.GetContextKeysCopy(nil)[0].Id != sub1.GetContextKeysCopy(nil)[1].Id {
		t.Error("root context keys must be identical")
	}

	sub2 := sub1.PushNewExecutionContext()
	if len(sub2.GetContextKeysCopy(nil)) != 3 {
		t.Error("non-root context must have 3 keys, the root and sub context key, and a new context key")
	}

	if sub1.GetContextKeysCopy(nil)[1].Id != sub2.GetContextKeysCopy(nil)[2].Id {
		t.Error("root context keys must be identical")
	}

	if sub1.GetContextKeysCopy(nil)[0].Id != sub2.GetContextKeysCopy(nil)[1].Id {
		t.Error("non-root context keys must be identical")
	}

	if sub1.GetLastContextKey().Id == sub2.GetLastContextKey().Id {
		t.Error("non-root context keys must be different")
	}
}
