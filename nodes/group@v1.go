package nodes

import (
	"actionforge/graph-runner/core"
	"bytes"
	_ "embed"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed group@v1.yml
var subgraphDefinition string

var DefaultExec core.OutputId = "exec"

type GroupNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *GroupNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *GroupNode) ExecuteImpl(c core.ExecutionContext) error {
	err := n.Execute(n.GetExecutionPort("exec"), c)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	err := core.RegisterNodeFactory(subgraphDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {

		var yamlDef bytes.Buffer
		err := yaml.NewEncoder(&yamlDef).Encode(nodeDef["graph"])
		if err != nil {
			return nil, err
		}

		ag, err := core.LoadGraph(yamlDef.Bytes())
		if err != nil {
			return nil, err
		}

		group := GroupNode{
			NodeBaseComponent: core.NodeBaseComponent{
				Subgraph: &ag,
			},
		}

		if ag.Inputs != nil {
			group.SetInputDefs(ag.Inputs)

			groupStart, err := ag.FindNode(ag.Entry)
			if err != nil {
				return nil, errors.New("group has no entry")
			}

			groupStartOutputs, ok := groupStart.(core.HasInputsInterface)
			if ok {
				for k := range ag.Inputs {
					groupStartOutputs.ConnectDataPort(k, core.DataSource{
						SrcNode: &group,
						Output:  core.OutputId(k),
					})
				}
				groupStartOutputs.SetInputDefs(ag.Inputs)
			}
		}

		if ag.Outputs != nil {
			group.SetOutputDefs(ag.Outputs)

			var groupEnd core.NodeRef
			for _, node := range ag.GetNodes() {
				if strings.HasPrefix(node.GetNodeType(), "group-output@") {
					groupEnd = node
					break
				}
			}

			groupEndInputs, ok := groupEnd.(core.HasOutputsInterface)
			if ok {
				for k := range ag.Outputs {
					group.ConnectDataPort(core.InputId(k), core.DataSource{
						SrcNode: groupEndInputs,
						Output:  k,
					})
				}
			}
		}

		return &group, nil
	})
	if err != nil {
		panic(err)
	}
}
