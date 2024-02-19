package cmd

import (
	"actionforge/graph-runner/core"
	"actionforge/graph-runner/utils"
	u "actionforge/graph-runner/utils"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	// initialize all nodes
	_ "actionforge/graph-runner/nodes"
)

var cmdRoot = &cobra.Command{
	Use:     "graph-runner [filename]",
	Short:   "Graph runner is a tool for running action graphs.",
	Version: core.GetFulllVersionInfo(),
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

func Execute() {
	_ = cmdRoot.Flags().Parse(os.Args[1:])

	var filename string
	if len(os.Args) > 1 {
		if strings.HasSuffix(os.Args[1], ".yml") {
			filename = os.Args[1]
		}
	}

	if filename == "" {
		filename = u.GetVariable("graph_file", "The graph file to use", u.GetVariableOpts{
			Env:      true,
			Optional: true,
		})
	}

	if filename != "" {
		err := ExecuteRun(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	} else {
		err := cmdRoot.Execute()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func ExecuteRun(graphFile string) error {

	graphContent, err := os.ReadFile(graphFile)
	if err != nil {
		return err
	}

	err = core.RunGraph(graphContent)
	if err != nil {
		return err
	}

	return nil
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
