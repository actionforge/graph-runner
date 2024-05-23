//go:build !github_impl

package utils

import (
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	onceVarsRe sync.Once
	varsRe     *regexp.Regexp
)

var (
	// This is the map of secrets that are available during the execution
	// of the action graph. The values contain the context name and
	// the secret value. Example: 'secrets.input1'
	secrets = make(map[string]string, 0)
)

func AddSecret(name string, secret string) {
	secrets[name] = secret
}

func GetSecret(name string) (string, bool) {
	secret, ok := secrets[name]
	return secret, ok
}

func RemoveSecret(name string) {
	delete(secrets, name)
}

func getGithubVarsRe() *regexp.Regexp {
	onceVarsRe.Do(func() {
		varsRe = regexp.MustCompile(`\$\{\{\s*(env|github|matrix|inputs|secrets)\.[\w]+\s*\}\}`)
	})
	return varsRe
}

func ReplaceContextVariables(input string) string {

	return getGithubVarsRe().ReplaceAllStringFunc(input, func(s string) string {
		// Remove the template syntax to get the context variable
		contextVar := strings.Trim(s, "${ }")

		if strings.HasPrefix(contextVar, "env.") {
			envVar, exists := os.LookupEnv(strings.TrimPrefix(contextVar, "env."))
			if exists {
				return envVar
			}
			return ""
		} else if strings.HasPrefix(contextVar, "secrets.") {
			secretVal, exists := secrets[contextVar]
			if exists {
				return secretVal
			}
			return ""
		}

		// If the context variable is not found, return the original string
		// Should never happen as the regex should only match the context variables above.
		return s
	})
}

func init() {
	LoadEnvOnce()
}
