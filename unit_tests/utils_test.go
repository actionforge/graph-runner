//go:build unit_tests

package unit_tests

import (
	"actionforge/graph-runner/utils"
	"testing"
)

func Test_If(t *testing.T) {

	r1 := utils.If(true, 1, 2)
	if r1 != 1 {
		t.Error("If(true, 1, 2) must be 1")
	}

	r2 := utils.If(false, 1, 2)
	if r2 != 2 {
		t.Error("If(false, 1, 2) must be 2")
	}

	r3 := utils.If(true, "a", "b")
	if r3 != "a" {
		t.Error("If(true, \"a\", \"b\") must be \"a\"")
	}

	r4 := utils.If(false, "a", "b")
	if r4 != "b" {
		t.Error("If(false, \"a\", \"b\") must be \"b\"")
	}
}
