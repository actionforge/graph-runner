package nodes

import (
	"actionforge/graph-runner/core"
	_ "embed"
)

//go:embed test@v1.yml
var testNodeDefinition string

type TestNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
}

func init() {
	err := core.RegisterNodeFactory(testNodeDefinition, func(context interface{}) (core.NodeRef, error) {
		return &TestNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
