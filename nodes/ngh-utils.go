//go:build !github_impl

package nodes

// Dummy implementation if project isn't build with GitHub support
func ReplaceContextVariables(input string) string {
	return input
}
