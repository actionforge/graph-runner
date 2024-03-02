package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	"bytes"
	_ "embed"
	"errors"

	"gopkg.in/yaml.v2"
)

//go:embed subgraph@v1.yml
var subgraphDefinition string

type SubGraphNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	ag    core.ActionGraph
	start core.NodeExecutionInterface
}

func (n *SubGraphNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *SubGraphNode) ExecuteImpl(c core.ExecutionContext) error {
	err := n.Execute(n.Executions[ni.Subgraph_v1_Output_exec], c)
	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func anyToPortDefinition[T any](o any) (T, error) {
	var (
		tmp bytes.Buffer
		ret T
	)
	err := yaml.NewEncoder(&tmp).Encode(o)
	if err != nil {
		return ret, err
	}

	err = yaml.NewDecoder(&tmp).Decode(&ret)
	if err != nil {
		return ret, err
	}
	return ret, err
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

		subStart, err := ag.FindNode(ag.Entry)
		if err != nil {
			return nil, errors.New("subgraph has no entry")
		}

		subStartExec, ok := subStart.(core.NodeExecutionInterface)
		if !ok {
			return nil, errors.New("subgraph entry is not an executable node")
		}

		subgraph := SubGraphNode{
			ag:    ag,
			start: subStartExec,
		}

		subgraph.Executions = make(map[core.OutputId]core.NodeExecutionInterface)
		subgraph.Executions[ni.Subgraph_v1_Output_exec] = subStartExec

		inputs, ok := nodeDef["inputs"]
		if ok {
			idefs := make(map[core.InputId]core.InputDefinition)
			odefs := make(map[core.OutputId]core.OutputDefinition)
			for k, v := range inputs.(map[any]any) {
				idef, err := anyToPortDefinition[core.InputDefinition](v)
				if err != nil {
					return nil, err
				}

				odef, err := anyToPortDefinition[core.OutputDefinition](v)
				if err != nil {
					return nil, err
				}

				idefs[core.InputId(k.(string))] = idef
				odefs[core.OutputId(k.(string))] = odef
			}
			subgraph.SetInputDefs(idefs)
			subgraph.SetOutputDefs(odefs)

			subStartInputs, ok := subStart.(core.HasInputsInterface)
			if ok {
				for k := range idefs {
					subStartInputs.ConnectDataPort(k, core.SourceNode{
						Name: core.OutputId(k),
						Src:  &subgraph,
					})
				}
				subStartInputs.SetInputDefs(idefs)
			}

			subStartOutputs, ok := subStart.(core.HasOutputsInterface)
			if ok {
				subStartOutputs.SetOutputDefs(odefs)
			}
		}

		return &subgraph, nil
	})
	if err != nil {
		panic(err)
	}
}
