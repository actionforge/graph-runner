//go:build github_impl

package nodes

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestReplaceContextVariables(t *testing.T) {
	// Fill github context variables with some dummy data

	// GITHUB_ACTIONS must be set in order to initialize the github context
	// structures in the init() function of gh-action.
	os.Setenv("GITHUB_ACTIONS", "true")

	os.Setenv("GITHUB_ACTOR", "actioncat")
	os.Setenv("GITHUB_BASE_REF", "main")
	os.Setenv("GITHUB_EVENT_NAME", "push")
	os.Setenv("GITHUB_JOB", "build")
	os.Setenv("GITHUB_EVENT_PATH", "/home/runner/work/_temp/_github_workflow/event.json")
	os.Setenv("GITHUB_HEAD_REF", "feature-branch")
	os.Setenv("GITHUB_REF", "refs/heads/main")
	os.Setenv("GITHUB_SHA", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0")
	os.Setenv("GITHUB_REPOSITORY", "my-user/my-repo")
	os.Setenv("GITHUB_WORKFLOW", "CI Build")

	os.Setenv("INPUT_TOKEN", "ghp_1234567890abcdef1234567890abcdef12345678")

	inputs := make(map[string]string)
	inputs["var123"] = "NumberedValue"
	inputs["custom"] = "CustomValue"
	inputs["emptyVar"] = ""
	inputs["var_name"] = "UnderValue"
	inputs["dup"] = "Duplicate"
	inputs["   "] = "SpaceVar"
	inputs["verylongvariablenameindeed"] = "LongValue"

	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(inputs)
	os.Setenv("INPUT_INPUTS", buffer.String())

	// Since the env variables are not coming from the parent
	// process and were instead set manually above, the github
	// context variables need to be initialized again.
	err := initGhContexts()
	if err != nil {
		t.Errorf("Error initializing github context: %s", err)
		return
	}

	// Define test cases
	testCases := []struct {
		name           string
		input          string
		expectedOutput string
	}{
		{
			name:           "Check github.repository",
			input:          "Repository is ${{ github.repository }}.",
			expectedOutput: "Repository is my-user/my-repo.",
		},
		{
			name:           "Check github.job",
			input:          "Job name: ${{ github.job }}.",
			expectedOutput: "Job name: build.",
		},
		{
			name:           "Check github.ref",
			input:          "Reference: ${{ github.ref }}.",
			expectedOutput: "Reference: refs/heads/main.",
		},
		{
			name:           "Check github.sha",
			input:          "Commit SHA: ${{ github.sha }}.",
			expectedOutput: "Commit SHA: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0.",
		},
		{
			name:           "Check github.event",
			input:          "Event name: ${{ github.event_name }}.",
			expectedOutput: "Event name: push.",
		},
		{
			name:           "Variable with number",
			input:          "Number test: ${{ inputs.var123 }}.",
			expectedOutput: "Number test: NumberedValue.",
		},
		{
			name:           "Multiple GitHub variables",
			input:          "${{ github.job }} and ${{ github.workflow }}.",
			expectedOutput: "build and CI Build.",
		},
		{
			name:           "Mixed GitHub and custom variables",
			input:          "Repo: ${{ github.repository }}, Custom: ${{ inputs.custom }}.",
			expectedOutput: "Repo: my-user/my-repo, Custom: CustomValue.",
		},
		{
			name:           "Variable at start",
			input:          "${{ github.workflow }} - workflow",
			expectedOutput: "CI Build - workflow",
		},
		{
			name:           "Variable at end",
			input:          "End with - ${{ github.job }}",
			expectedOutput: "End with - build",
		},
		{
			name:           "Multiple adjacent variables",
			input:          "${{ github.job }}${{ github.workflow }}",
			expectedOutput: "buildCI Build",
		},
		{
			name:           "Empty variable",
			input:          "This is empty: ${{ inputs.emptyVar }}.",
			expectedOutput: "This is empty: .",
		},
		{
			name:           "Undefined variable",
			input:          "This is ${{ inputs.undefined }}.",
			expectedOutput: "This is .",
		},
		{
			name:           "Variable with underscore",
			input:          "Underscore: ${{ inputs.var_name }}.",
			expectedOutput: "Underscore: UnderValue.",
		},
		{
			name:           "Nonexistent GitHub context variable",
			input:          "Nonexistent ${{ github.nonexistent }}.",
			expectedOutput: "Nonexistent .",
		},
		{
			name:           "Multiple replacements with same key",
			input:          "${{ inputs.dup }} and ${{ inputs.dup }} again.",
			expectedOutput: "Duplicate and Duplicate again.",
		},
		{
			name:           "Variable with only spaces (spaces must be ignored)",
			input:          "Just spaces: ${{ inputs.   }}.",
			expectedOutput: "Just spaces: ${{ inputs.   }}.",
		},
		{
			name:           "Long variable name",
			input:          "Long variable: ${{ inputs.verylongvariablenameindeed }}.",
			expectedOutput: "Long variable: LongValue.",
		},
		{
			name:           "Check github.token",
			input:          "Secret is ${{ github.token }}.",
			expectedOutput: "Secret is ghp_1234567890abcdef1234567890abcdef12345678.",
		},
		{
			name:           "Check github.token",
			input:          "Secret is ${{ secrets.GITHUB_TOKEN }}.",
			expectedOutput: "Secret is ghp_1234567890abcdef1234567890abcdef12345678.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output := ReplaceContextVariables(tc.input)
			if output != tc.expectedOutput {
				t.Errorf("Output should match expected output, but got: %s", output)
			}
		})
	}
}
