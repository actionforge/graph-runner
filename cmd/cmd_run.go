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
	Use:   "run [filename]",
	Short: `Run a graph file`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[1]
		if filename == "" {
			filename = u.GetVariable("graph_file", "The graph file to use", u.GetVariableOpts{
				Env: true,
			})

			if filename == "" {
				fmt.Println("argument [filename] or environment variable 'GRAPH_FILE' is required")
				os.Exit(1)
			}
		}

		err := ExecuteRun(filename)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		os.Exit(0)
	},
}

func init() {
	cmdRoot.AddCommand(cmdRun)
}
