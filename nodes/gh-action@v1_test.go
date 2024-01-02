//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	"os"
	"testing"
)

func TestReplaceContextVariables(t *testing.T) {

	// Fill github context variables with some dummy data

	os.Setenv("GITHUB_ACTOR", "actioncat")
	os.Setenv("GITHUB_BASE_REF", "main")
	os.Setenv("GITHUB_EVENT_NAME", "push")
	os.Setenv("GITHUB_JOB", "build")
	os.Setenv("GITHUB_EVENT_PATH", "/home/runner/work/_temp/_github_workflow/event.json")
	os.Setenv("GITHUB_HEAD_REF", "feature-branch")
	os.Setenv("GITHUB_REF", "refs/heads/main")
	os.Setenv("GITHUB_SHA", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0")
	os.Setenv("GITHUB_REPOSITORY", "my-user/my-repo")
	os.Setenv("GITHUB_TOKEN", "ghp_1234567890abcdef1234567890abcdef12345678")
	os.Setenv("GITHUB_WORKFLOW", "CI Build")

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
			input:          "Event name: ${{ github.event_name }}.",
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
