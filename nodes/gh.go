//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"os"
	"strings"
)

var (
	// This is the token of `ACTIONS_RUNTIME_TOKEN` that is used to
	// communicate with certain github APIs, like when github actions
	// are pulled from github.com.
	ghActionsRuntimeToken string

	// This is the map of secrets that are available during the execution
	// of the action graph. The values contain the context name and
	// the secret value. Example: secrets.input1 github.token
	ghSecrets = make(map[string]string, 0)

	// this is the map of context variables that are available to the action
	ghContext = make(map[string]string)
)

func AddGhSecret(name string, secret string) {
	ghSecrets[name] = secret
}

func RemoveGhSecret(name string) {
	delete(ghSecrets, name)
}

func initGhContexts() {

	ghContext["github.workspace"] = os.Getenv("GITHUB_WORKSPACE")
	ghContext["github.repository"] = os.Getenv("GITHUB_REPOSITORY")
	ghContext["github.job"] = os.Getenv("GITHUB_JOB")
	ghContext["github.ref"] = os.Getenv("GITHUB_REF")
	ghContext["github.sha"] = os.Getenv("GITHUB_SHA")
	ghContext["github.event_name"] = os.Getenv("GITHUB_EVENT_NAME")
	ghContext["github.workflow"] = os.Getenv("GITHUB_WORKFLOW")

	// As outlined in the documentation, secrets.GITHUB_TOKEN and github.token
	// are functionally equivalent. See:
	// https://docs.github.com/en/actions/learn-github-actions/contexts#github-context
	ghContext["github.token"] = os.Getenv("INPUT_TOKEN")
	ghSecrets["secrets.GITHUB_TOKEN"] = os.Getenv("INPUT_TOKEN")

	ghActionsRuntimeToken = os.Getenv("ACTIONS_RUNTIME_TOKEN")

	for _, env := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(env), "SECRET_") {
			pair := strings.SplitN(env, "=", 2)
			if len(pair) == 1 {
				// empty secrets are valid
				ghSecrets[pair[0]] = ""
			} else if len(pair) == 2 {
				key := strings.TrimPrefix(strings.ToUpper(pair[0]), "SECRET_")
				value := pair[1]

				ghSecrets[key] = value
				os.Unsetenv(pair[0])
			} else {
				fmt.Println("WARN: Invalid secret: ", pair[0])
			}
		}
	}
}

func init() {

	utils.LoadEnvOnce()

	initGhContexts()
}
