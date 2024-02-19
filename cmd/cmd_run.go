package cmd

import (
	"actionforge/graph-runner/core"
	u "actionforge/graph-runner/utils"
	"fmt"
	"os"

	// initialize all nodes

	_ "actionforge/graph-runner/nodes"

	_ "embed"

	"github.com/spf13/cobra"
)

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

var cmdRun = &cobra.Command{
	Use:   "run",
	Short: `Run a graph file`,
	Run: func(cmd *cobra.Command, args []string) {

		graphFile, _ := cmd.Flags().GetString("graph_file")
		if graphFile == "" {
			graphFile = u.GetVariable("graph_file", "The graph file to use", u.GetVariableOpts{
				Env: true,
			})

			if graphFile == "" {
				fmt.Println("--graph_file or env:GRAPH_FILE is required")
				os.Exit(1)
			}
		}

		err := ExecuteRun(graphFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

func init() {
	cmdRoot.AddCommand(cmdRun)

	cmdRun.Flags().String("graph_file", "", "The graph file to run")
}
