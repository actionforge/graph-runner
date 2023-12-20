package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var cmdRoot = &cobra.Command{
	Use:     "graph-runner",
	Short:   "Graph runner is a tool for running action graphs.",
	Version: "0.1a",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	_ = cmdRoot.PersistentFlags().Parse(os.Args[1:])

	_ = cmdRoot.Execute()
}
