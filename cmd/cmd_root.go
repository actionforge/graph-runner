package cmd

import (
	"actionforge/graph-runner/core"
	"os"

	"github.com/spf13/cobra"
)

var cmdRoot = &cobra.Command{
	Use:     "graph-runner",
	Short:   "Graph runner is a tool for running action graphs.",
	Version: core.GetFulllVersionInfo(),
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	_ = cmdRoot.PersistentFlags().Parse(os.Args[1:])

	_ = cmdRoot.Execute()
}
