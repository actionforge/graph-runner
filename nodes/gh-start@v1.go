//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	"context"
	_ "embed"
	"fmt"
	"os"
)

//go:embed gh-start@v1.yml
var GithubActionStartNodeDefinition string

type GhActionStartNode struct {
	core.NodeBaseComponent
	core.Executions
}

const unexpectedEventErrorStr = `
Error: No trigger port connected for event: '%s'

For more information, verify the accepted trigger events in
your GitHub Action workflow file and consult the documentation:
🔗 https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#%s`

func (n *GhActionStartNode) ExecuteEntry(inputValues map[core.OutputId]any) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := core.NewExecutionContext(ctx)
	exec, err := n.GetStartOutput()
	if err != nil {
		return err
	}
	return n.Execute(exec, c)
}

func (n *GhActionStartNode) ExecuteImpl(c core.ExecutionContext, inputId core.InputId) error {
	exec, err := n.GetStartOutput()
	if err != nil {
		return err
	}

	err = n.Execute(exec, c)
	if err != nil {
		return err
	}

	return nil
}

func (n *GhActionStartNode) GetStartOutput() (core.OutputId, error) {
	event := os.Getenv("GITHUB_EVENT_NAME")

	var exec core.OutputId

	// All trigger events are listed here:
	// https://docs.github.com/en/actions/reference/events-that-trigger-workflows
	switch event {
	case "branch_protection_rule":
		exec = ni.Gh_start_v1_Output_exec_on_branch_protection_rule
	case "check_run":
		exec = ni.Gh_start_v1_Output_exec_on_check_run
	case "check_suite":
		exec = ni.Gh_start_v1_Output_exec_on_check_suite
	case "create":
		exec = ni.Gh_start_v1_Output_exec_on_create
	case "delete":
		exec = ni.Gh_start_v1_Output_exec_on_delete
	case "deployment":
		exec = ni.Gh_start_v1_Output_exec_on_deployment
	case "deployment_status":
		exec = ni.Gh_start_v1_Output_exec_on_deployment_status
	case "discussion":
		exec = ni.Gh_start_v1_Output_exec_on_discussion
	case "discussion_comment":
		exec = ni.Gh_start_v1_Output_exec_on_discussion_comment
	case "fork":
		exec = ni.Gh_start_v1_Output_exec_on_fork
	case "gollum":
		exec = ni.Gh_start_v1_Output_exec_on_gollum
	// it looks like pull_request_comment is deprecated and substituted with 'issue_comment'
	case "issue_comment", "pull_request_comment":
		exec = ni.Gh_start_v1_Output_exec_on_issue_comment
	case "issues":
		exec = ni.Gh_start_v1_Output_exec_on_issues
	case "label":
		exec = ni.Gh_start_v1_Output_exec_on_label
	case "merge_group":
		exec = ni.Gh_start_v1_Output_exec_on_merge_group
	case "milestone":
		exec = ni.Gh_start_v1_Output_exec_on_milestone
	case "page_build":
		exec = ni.Gh_start_v1_Output_exec_on_page_build
	case "project":
		exec = ni.Gh_start_v1_Output_exec_on_project
	case "project_card":
		exec = ni.Gh_start_v1_Output_exec_on_project_card
	case "project_column":
		exec = ni.Gh_start_v1_Output_exec_on_project_column
	case "public":
		exec = ni.Gh_start_v1_Output_exec_on_public
	case "pull_request":
		exec = ni.Gh_start_v1_Output_exec_on_pull_request
	case "pull_request_review":
		exec = ni.Gh_start_v1_Output_exec_on_pull_request_review
	case "pull_request_review_comment":
		exec = ni.Gh_start_v1_Output_exec_on_pull_request_review_comment
	case "pull_request_target":
		exec = ni.Gh_start_v1_Output_exec_on_pull_request_target
	case "push":
		exec = ni.Gh_start_v1_Output_exec_on_push
	case "registry_package":
		exec = ni.Gh_start_v1_Output_exec_on_registry_package
	case "release":
		exec = ni.Gh_start_v1_Output_exec_on_release
	case "repository_dispatch":
		exec = ni.Gh_start_v1_Output_exec_on_repository_dispatch
	case "schedule":
		exec = ni.Gh_start_v1_Output_exec_on_schedule
	case "status":
		exec = ni.Gh_start_v1_Output_exec_on_status
	case "watch":
		exec = ni.Gh_start_v1_Output_exec_on_watch
	case "workflow_call":
		exec = ni.Gh_start_v1_Output_exec_on_workflow_call
	case "workflow_dispatch":
		exec = ni.Gh_start_v1_Output_exec_on_workflow_dispatch
	case "workflow_run":
		exec = ni.Gh_start_v1_Output_exec_on_workflow_run
	default:
		return "", fmt.Errorf("unknown event name: %s", event)
	}

	_, ok := n.GetExecutionTarget(exec)
	if !ok {
		return "", fmt.Errorf(unexpectedEventErrorStr, event, event)
	}

	return exec, nil
}

func init() {
	utils.SetFeature("github", true)

	err := core.RegisterNodeFactory(GithubActionStartNodeDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &GhActionStartNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
