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
	return n.Execute(n, c)
}

func (n *GhActionStartNode) ExecuteImpl(c core.ExecutionContext) error {

	event := os.Getenv("GITHUB_EVENT_NAME")

	var exec core.NodeExecutionInterface

	// All trigger events are listed here:
	// https://docs.github.com/en/actions/reference/events-that-trigger-workflows
	switch event {
	case "branch_protection_rule":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_branch_protection_rule)
	case "check_run":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_check_run)
	case "check_suite":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_check_suite)
	case "create":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_create)
	case "delete":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_delete)
	case "deployment":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_deployment)
	case "deployment_status":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_deployment_status)
	case "discussion":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_discussion)
	case "discussion_comment":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_discussion_comment)
	case "fork":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_fork)
	case "gollum":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_gollum)
	// it looks like pull_request_comment is deprecated and substituted with 'issue_comment'
	case "issue_comment", "pull_request_comment":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_issue_comment)
	case "issues":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_issues)
	case "label":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_label)
	case "merge_group":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_merge_group)
	case "milestone":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_milestone)
	case "page_build":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_page_build)
	case "project":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_project)
	case "project_card":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_project_card)
	case "project_column":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_project_column)
	case "public":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_public)
	case "pull_request":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_pull_request)
	case "pull_request_review":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_pull_request_review)
	case "pull_request_review_comment":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_pull_request_review_comment)
	case "pull_request_target":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_pull_request_target)
	case "push":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_push)
	case "registry_package":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_registry_package)
	case "release":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_release)
	case "repository_dispatch":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_repository_dispatch)
	case "schedule":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_schedule)
	case "status":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_status)
	case "watch":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_watch)
	case "workflow_call":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_workflow_call)
	case "workflow_dispatch":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_workflow_dispatch)
	case "workflow_run":
		exec = n.GetTargetNode(ni.Gh_start_v1_Output_exec_on_workflow_run)
	default:
		return fmt.Errorf("unknown event name: %s", event)
	}

	if exec == nil {
		return fmt.Errorf(unexpectedEventErrorStr, event, event)
	}

	err := n.Execute(exec, c)
	if err != nil {
		return err
	}

	return nil
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
