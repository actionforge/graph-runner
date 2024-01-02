package cmd

import (
	u "actionforge/graph-runner/utils"
	"fmt"
	"os"

	// initialize all nodes
	"actionforge/graph-runner/core"
	_ "actionforge/graph-runner/nodes"

	"github.com/spf13/cobra"
)

func ExecuteRun(graphFile string) (error_code int, err error) {
	ag, err := core.LoadActionGraph(graphFile)
	if err != nil {
		return 1, err
	}

	entry, err := ag.GetEntry()
	if err != nil {
		return 2, err
	}

	err = entry.ExecuteEntry()
	if err != nil {
		return 3, err
	}

	return 0, nil
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

		exitCode, err := ExecuteRun(graphFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(exitCode)
		}

		os.Exit(0)
	},
}

func init() {
	cmdRoot.AddCommand(cmdRun)

	cmdRun.Flags().String("graph_file", "", "The graph file to run")
}
