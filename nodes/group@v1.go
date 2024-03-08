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

//go:embed group@v1.yml
var subgraphDefinition string

type GroupNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions

	ag    core.ActionGraph
	start core.NodeExecutionInterface
}

func (n *GroupNode) OutputValueById(c core.ExecutionContext, outputId core.OutputId) (interface{}, error) {

	v, err := n.InputValueById(c, core.InputId(outputId), nil)
	if err != nil {
		return nil, err
	}

	return v, nil
}

func (n *GroupNode) ExecuteImpl(c core.ExecutionContext) error {
	err := n.Execute(n.Executions[ni.Subgraph_v1_Output_exec], c)
	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(subgraphDefinition, func(ctx interface{}, nodeDef map[any]any) (core.NodeRef, error) {

		var yamlDef bytes.Buffer
		err := yaml.NewEncoder(&yamlDef).Encode(nodeDef["graph"])
		if err != nil {
			return nil, err
		}

		ag, err := core.LoadGraph(yamlDef.Bytes())
		if err != nil {
			return nil, err
		}

		subStart, err := ag.FindNode(ag.Entry)
		if err != nil {
			return nil, errors.New("group has no entry")
		}

		subStartExec, ok := subStart.(core.NodeExecutionInterface)
		if !ok {
			return nil, errors.New("group entry is not an executable node")
		}

		group := GroupNode{
			ag:    ag,
			start: subStartExec,
		}

		group.Executions = make(map[core.OutputId]core.NodeExecutionInterface)
		group.Executions[ni.Subgraph_v1_Output_exec] = subStartExec

		if ag.Inputs != nil {
			group.SetInputDefs(ag.Inputs)

			subStartInputs, ok := subStart.(core.HasInputsInterface)
			if ok {
				for k := range ag.Inputs {
					subStartInputs.ConnectDataPort(k, core.DataSource{
						Output:  core.OutputId(k),
						SrcNode: &group,
					})
				}
				subStartInputs.SetInputDefs(ag.Inputs)
			}
		}

		if ag.Outputs != nil {
			group.SetOutputDefs(ag.Outputs)

			groupStartOutputs, ok := subStart.(core.HasOutputsInterface)
			if ok {
				// TODO: (Seb)
				/*
					for k := range ag.Inputs {
						subStartInputs.ConnectDataPort(k, core.DataSource{
							Output:  core.OutputId(k),
							SrcNode: &group,
						})
					}
				*/
				groupStartOutputs.SetOutputDefs(ag.Outputs)
			}
		}

		return &group, nil
	})
	if err != nil {
		panic(err)
	}
}
