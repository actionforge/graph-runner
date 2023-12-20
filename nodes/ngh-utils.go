//go:build !github_impl
// +build !github_impl

package nodes

import "actionforge/graph-runner/core"

// Dummy implementation if project isn't build with GitHub support
func ReplaceContextVariables(input string, inputValues map[core.InputId]interface{}) string {
	return input
}
