package nodes

import (
	"actionforge/graph-runner/core"
	"bytes"
	_ "embed"

	"gopkg.in/yaml.v2"
)

//go:embed subgraph@v1.yml
var subgraphDefinition string

type SubGraphNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	ag core.ActionGraph
}

func (n *SubGraphNode) ExecuteImpl(c core.ExecutionContext) error {
	entry, err := n.ag.GetEntry()
	if err != nil {
		return err
	}

	err = entry.ExecuteEntry()
	if err != nil {
		return err
	}
	return nil
}

func init() {
	err := core.RegisterNodeFactory(subgraphDefinition, func(ctx interface{}, nodeDef map[any]any) (core.NodeRef, error) {

		var yamlDef bytes.Buffer
		err := yaml.NewEncoder(&yamlDef).Encode(nodeDef)
		if err != nil {
			return nil, err
		}

		ag, err := core.LoadGraph(yamlDef.Bytes())
		if err != nil {
			return nil, err
		}

		return &SubGraphNode{
			ag: ag,
		}, nil
	})
	if err != nil {
		panic(err)
	}
}
