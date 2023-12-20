package cmd

import (
	u "actionforge/graph-runner/utils"
	"flag"
	"fmt"
	"os"
	"strings"

	// initialize all nodes
	"actionforge/graph-runner/core"
	_ "actionforge/graph-runner/nodes"

	"github.com/joho/godotenv"
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

func init() {
	loadEnvFile := os.Getenv("LOAD_ENV_FILE")
	if loadEnvFile != "" {
		_ = godotenv.Load()
	}

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

	// Extract secrets and github tokens if available
	if os.Getenv("GITHUB_ACTIONS") == "true" {

		githubToken := os.Getenv("ACTIONS_RUNTIME_TOKEN")
		if githubToken == "" {
			panic("ACTIONS_RUNTIME_TOKEN is empty")
		}

		core.G_githubToken = githubToken

		for _, env := range os.Environ() {
			if strings.HasPrefix(strings.ToUpper(env), "SECRET_") {
				pair := strings.SplitN(env, "=", 2)
				if len(pair) == 1 {
					// empty secrets are valid
					core.G_secrets[pair[0]] = ""
				} else if len(pair) == 2 {
					key := strings.TrimPrefix(strings.ToUpper(pair[0]), "SECRET_")
					value := pair[1]

					core.G_secrets[key] = value
					os.Unsetenv(pair[0])
				} else {
					fmt.Println("WARN: Invalid secret: ", pair[0])
				}
			}
		}

		githubRepo := os.Getenv("GITHUB_REPOSITORY")
		if githubRepo == "" {
			panic("GITHUB_REPOSITORY is empty")
		}

		githubRepoPair := strings.Split(githubRepo, "/")
		if githubRepoPair == nil || len(githubRepoPair) != 2 {
			panic("GITHUB_REPOSITORY expected to be in the format of 'org/repo'")
		}

		core.G_repositoryOrg = githubRepoPair[0]
		core.G_repositoryName = githubRepoPair[1]
	}
}
