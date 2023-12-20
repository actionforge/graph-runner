//go:build github_impl
// +build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	"testing"
)

func TestReplaceContextVariables(t *testing.T) {

	// Fill github context variables with some dummy data
	githubContext = map[string]string{
		"github.workspace":  "/home/runner/work/my-repo/my-repo",
		"github.repository": "my-user/my-repo",
		"github.job":        "build",
		"github.ref":        "refs/heads/main",
		"github.sha":        "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0",
		"github.event":      "push",
		"github.workflow":   "CI Build",

		"github.token": "ghp_1234567890abcdef1234567890abcdef12345678",
	}

	// Define test cases
	testCases := []struct {
		name           string
		input          string
		inputValues    map[core.InputId]interface{}
		expectedOutput string
	}{
		{
			name:           "Check github.repository",
			input:          "Repository is ${{ github.repository }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Repository is my-user/my-repo.",
		},
		{
			name:           "Check github.job",
			input:          "Job name: ${{ github.job }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Job name: build.",
		},
		{
			name:           "Check github.ref",
			input:          "Reference: ${{ github.ref }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Reference: refs/heads/main.",
		},
		{
			name:           "Check github.sha",
			input:          "Commit SHA: ${{ github.sha }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Commit SHA: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0.",
		},
		{
			name:           "Check github.event",
			input:          "Event name: ${{ github.event }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Event name: push.",
		},
		{
			name:  "Variable with number",
			input: "Number test: ${{ inputs.var123 }}.",
			inputValues: map[core.InputId]interface{}{
				"var123": "NumberedValue",
			},
			expectedOutput: "Number test: NumberedValue.",
		},
		{
			name:           "Multiple GitHub variables",
			input:          "${{ github.job }} and ${{ github.workflow }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "build and CI Build.",
		},
		{
			name:  "Mixed GitHub and custom variables",
			input: "Repo: ${{ github.repository }}, Custom: ${{ inputs.custom }}.",
			inputValues: map[core.InputId]interface{}{
				"custom": "CustomValue",
			},
			expectedOutput: "Repo: my-user/my-repo, Custom: CustomValue.",
		},
		{
			name:           "Variable at start",
			input:          "${{ github.workflow }} - workflow",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "CI Build - workflow",
		},
		{
			name:           "Variable at end",
			input:          "End with - ${{ github.job }}",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "End with - build",
		},
		{
			name:           "Multiple adjacent variables",
			input:          "${{ github.job }}${{ github.workflow }}",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "buildCI Build",
		},
		{
			name:  "Empty variable",
			input: "This is empty: ${{ inputs.emptyVar }}.",
			inputValues: map[core.InputId]interface{}{
				"emptyVar": "",
			},
			expectedOutput: "This is empty: .",
		},
		{
			name:           "Undefined variable",
			input:          "This is ${{ inputs.undefined }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "This is .",
		},
		{
			name:  "Variable with underscore",
			input: "Underscore: ${{ inputs.var_name }}.",
			inputValues: map[core.InputId]interface{}{
				"var_name": "UnderValue",
			},
			expectedOutput: "Underscore: UnderValue.",
		},
		{
			name:           "Nonexistent GitHub context variable",
			input:          "Nonexistent ${{ github.nonexistent }}.",
			inputValues:    map[core.InputId]interface{}{},
			expectedOutput: "Nonexistent .",
		},
		{
			name:  "Multiple replacements with same key",
			input: "${{ inputs.dup }} and ${{ inputs.dup }} again.",
			inputValues: map[core.InputId]interface{}{
				"dup": "Duplicate",
			},
			expectedOutput: "Duplicate and Duplicate again.",
		},
		{
			name:  "Variable with only spaces (spaces must be ignored)",
			input: "Just spaces: ${{ inputs.   }}.",
			inputValues: map[core.InputId]interface{}{
				"   ": "SpaceVar",
			},
			expectedOutput: "Just spaces: ${{ inputs.   }}.",
		},
		{
			name:  "Long variable name",
			input: "Long variable: ${{ inputs.verylongvariablenameindeed }}.",
			inputValues: map[core.InputId]interface{}{
				"verylongvariablenameindeed": "LongValue",
			},
			expectedOutput: "Long variable: LongValue.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := ReplaceContextVariables(tc.input, tc.inputValues)
			if output != tc.expectedOutput {
				t.Errorf("Output should match expected output")
			}
		})
	}
}
