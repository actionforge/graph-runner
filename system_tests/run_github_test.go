//go:build system_tests
// +build system_tests

package system_tests

import (
	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	"fmt"
	"testing"
)

type testCase struct {
	validator string
	secret    string
	success   bool
}

// Simple test of the for loop
func Test_Secret(t *testing.T) {
	defer utils.LoggerString.Clear()

	core.G_secrets["API_KEY_123"] = "THIS_IS_A_SECRET"
	defer delete(core.G_secrets, "API_KEY_123")

	// Test the run node, env node, and string format node.
	exitCode, err := runGraphFile("system_tests/test_secret.yml")
	if err != nil {
		t.Fatal(err)
	} else if exitCode != 0 {
		t.Fatal("exitCode != 0")
	}

	actual := utils.LoggerString.String()

	expectedString := `Execute 'start (start@v1)'
Execute 'run-v1-butterfly-gray-shark (run@v1)'
THIS_IS_A_SECRET
`

	if !diffStrings(actual, expectedString) {
		t.Fatal()
	}
}

func Test_StartAction(t *testing.T) {
	defer utils.LoggerString.Clear()

	for _, event := range githubEvents {

		utils.LoggerString.Clear()

		t.Setenv("GITHUB_EVENT_NAME", event)

		// Test the run node, env node, and string format node.
		exitCode, err := runGraphFile("system_tests/test_gh-start.yml")
		if err != nil {
			t.Fatal(err)
		} else if exitCode != 0 {
			t.Fatal("exitCode != 0")
		}

		actual := utils.LoggerString.String()

		expectedString := fmt.Sprintf(`Execute 'gh-start (gh-start@v1)'
Execute 'node-%s (run@v1)'
Triggered by %s
`, event, event)

		if !diffStrings(actual, expectedString) {
			t.Fatal()
		}
	}

}

var githubEvents = []string{
	"branch_protection_rule",
	"check_run",
	"check_suite",
	"create",
	"delete",
	"deployment",
	"deployment_status",
	"discussion",
	"discussion_comment",
	"fork",
	"gollum",
	"issue_comment",
	"issues",
	"label",
	"merge_group",
	"milestone",
	"page_build",
	"project",
	"project_card",
	"project_column",
	"public",
	"pull_request",
	"pull_request_review",
	"pull_request_review_comment",
	"pull_request_target",
	"push",
	"registry_package",
	"release",
	"repository_dispatch",
	"schedule",
	"status",
	"watch",
	"workflow_call",
	"workflow_dispatch",
	"workflow_run",
}
