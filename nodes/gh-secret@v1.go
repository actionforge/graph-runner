//go:build github_impl

package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	_ "embed"
	"fmt"
)

//go:embed gh-secret@v1.yml
var GithubActionSecretDefinition string

type GhSecretsNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func (n *GhSecretsNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	secretName, err := core.InputValueById[string](c, n.Inputs, ni.Gh_secret_v1_Input_name)
	if err != nil {
		return nil, err
	}

	prefix, err := core.InputValueById[string](c, n.Inputs, ni.Gh_secret_v1_Input_prefix)
	if err != nil {
		return nil, err
	}

	var secretValue string

	if secretName == "GITHUB_TOKEN" {
		secretValue = core.G_githubToken
	} else {
		var ok bool
		secretValue, ok = core.G_secrets[secretName]
		if !ok {
			// return an empty string if the secret is not found
			return "", nil
		}
	}

	return fmt.Sprintf("%s%s", prefix, secretValue), nil
}

func init() {
	err := core.RegisterNodeFactory(GithubActionSecretDefinition, func(context interface{}) (core.NodeRef, error) {
		return &GhSecretsNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
