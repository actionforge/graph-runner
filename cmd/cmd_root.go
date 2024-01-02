package cmd

import (
	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	"flag"
	"fmt"
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

func init() {
	flag.Usage = func() {
		fmt.Print("\n")
		fmt.Fprintf(os.Stderr, "Usage: %s", os.Args[0])
		flag.VisitAll(func(f *flag.Flag) {
			defValue := f.DefValue
			if defValue == "" {
				defValue = "'...'"
			}
			fmt.Fprintf(os.Stderr, " -%s=%s", f.Name, defValue)
		})
		fmt.Print("\n\n")
	}

	utils.LoadEnvOnce()
}
