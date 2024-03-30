//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/utils"
	"encoding/json"
	"fmt"
	"log"
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
	// the secret value. Example: 'secrets.input1' or 'secrets.GITHUB_TOKEN'
	ghSecrets = make(map[string]string, 0)

	// this is the map of 'github.xyz' context variables
	ghContext = make(map[string]string)

	// this is the map of all 'matrix.xyz' variables
	ghMatrix = make(map[string]string)

	// this is the map of all 'inputs.xyz' variables
	ghInputs = make(map[string]string)
)

func GetSecrets() map[string]string {
	return ghSecrets
}

func AddGhSecret(name string, secret string) {
	ghSecrets[name] = secret
}

func RemoveGhSecret(name string) {
	delete(ghSecrets, name)
}

func decodeJsonFromEnvValue(envValue string, prefix string) (map[string]string, error) {
	envMap := map[string]string{}
	tmp := map[string]string{}
	err := json.NewDecoder(strings.NewReader(envValue)).Decode(&tmp)
	if err != nil {
		return nil, err
	}
	for k, v := range tmp {
		envMap[fmt.Sprintf("%s.%s", prefix, k)] = v
	}
	return envMap, nil
}

func initGhContexts() error {

	// For more information on the githubs context, see:
	// https://docs.github.com/en/actions/learn-github-actions/contexts

	ghContext["github.action"] = os.Getenv("GITHUB_ACTION")
	// No direct mapping yet for 'github.action_path'
	ghContext["github.actor"] = os.Getenv("GITHUB_ACTOR")
	ghContext["github.actor_id"] = os.Getenv("GITHUB_ACTOR_ID")
	ghContext["github.api_url"] = os.Getenv("GITHUB_API_URL")
	ghContext["github.base_ref"] = os.Getenv("GITHUB_BASE_REF")
	ghContext["github.env"] = os.Getenv("GITHUB_ENV")
	ghContext["github.event_name"] = os.Getenv("GITHUB_EVENT_NAME")
	ghContext["github.event_path"] = os.Getenv("GITHUB_EVENT_PATH")
	ghContext["github.graphql_url"] = os.Getenv("GITHUB_GRAPHQL_URL")
	ghContext["github.head_ref"] = os.Getenv("GITHUB_HEAD_REF")
	ghContext["github.job"] = os.Getenv("GITHUB_JOB")
	ghContext["github.ref"] = os.Getenv("GITHUB_REF")
	ghContext["github.ref_name"] = os.Getenv("GITHUB_REF_NAME")
	ghContext["github.ref_protected"] = os.Getenv("GITHUB_REF_PROTECTED")
	ghContext["github.ref_type"] = os.Getenv("GITHUB_REF_TYPE")
	ghContext["github.repository"] = os.Getenv("GITHUB_REPOSITORY")
	ghContext["github.repository_id"] = os.Getenv("GITHUB_REPOSITORY_ID")
	ghContext["github.repository_owner"] = os.Getenv("GITHUB_REPOSITORY_OWNER")
	ghContext["github.repository_owner_id"] = os.Getenv("GITHUB_REPOSITORY_OWNER_ID")
	ghContext["github.run_attempt"] = os.Getenv("GITHUB_RUN_ATTEMPT")
	ghContext["github.run_id"] = os.Getenv("GITHUB_RUN_ID")
	ghContext["github.run_number"] = os.Getenv("GITHUB_RUN_NUMBER")
	ghContext["github.server_url"] = os.Getenv("GITHUB_SERVER_URL")
	ghContext["github.sha"] = os.Getenv("GITHUB_SHA")
	ghContext["github.workflow"] = os.Getenv("GITHUB_WORKFLOW")
	ghContext["github.workflow_ref"] = os.Getenv("GITHUB_WORKFLOW_REF")
	ghContext["github.workspace"] = os.Getenv("GITHUB_WORKSPACE")

	// As outlined in the documentation, secrets.GITHUB_TOKEN and github.token
	// are functionally equivalent. See:
	// https://docs.github.com/en/actions/learn-github-actions/contexts#github-context
	ghContext["github.token"] = os.Getenv("INPUT_TOKEN")
	ghSecrets["secrets.GITHUB_TOKEN"] = os.Getenv("INPUT_TOKEN")

	ghActionsRuntimeToken = os.Getenv("ACTIONS_RUNTIME_TOKEN")

	for _, env := range os.Environ() {

		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 {
			continue
		}

		envName := kv[0]
		envValue := kv[1]

		if strings.HasPrefix(envName, "SECRET_") {
			key := strings.TrimPrefix(envName, "SECRET_")
			ghSecrets[key] = envValue
			os.Unsetenv(envName)
		} else if envName == "INPUT_MATRIX" {
			var err error
			ghMatrix, err = decodeJsonFromEnvValue(envValue, "matrix")
			if err != nil {
				return err
			}
			os.Unsetenv(envName)
		} else if envName == "INPUT_INPUTS" {
			var err error
			ghInputs, err = decodeJsonFromEnvValue(envValue, "inputs")
			if err != nil {
				return err
			}
			os.Unsetenv(envName)
		}
	}

	// The information in the inputs context and github.event.inputs context is identical
	// except that the inputs context preserves Boolean values as Booleans instead of converting
	// them to strings. TODO: (Seb) Change the ghInputs to map[string]interface{} to preserve
	// the types for inputs.
	for k, v := range ghSecrets {
		ghContext[fmt.Sprintf("github.event.%s", k)] = v
	}

	return nil
}

func init() {

	utils.LoadEnvOnce()

	err := initGhContexts()
	if err != nil {
		log.Fatalf("Error initializing github context: %s", err)
	}
}
