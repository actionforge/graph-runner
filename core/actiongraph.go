package core

import (
	u "actionforge/graph-runner/utils"
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type ActionGraph struct {
	Nodes   map[string]NodeRef
	Inputs  map[InputId]InputDefinition   `yaml:"inputs" json:"inputs" bson:"inputs"`
	Outputs map[OutputId]OutputDefinition `yaml:"outputs" json:"outputs" bson:"outputs"`

	Entry string
}

func (ag *ActionGraph) AddNode(nodeId string, node NodeRef) {
	ag.Nodes[nodeId] = node
}

func (ag *ActionGraph) FindNode(nodeId string) (NodeRef, error) {
	node, exists := ag.Nodes[nodeId]
	if !exists {
		return nil, fmt.Errorf("node %s not found", nodeId)
	}
	return node, nil
}

func (ag *ActionGraph) GetNodes() map[string]NodeRef {
	return ag.Nodes
}

func (ag *ActionGraph) SetEntry(entryName string) {
	ag.Entry = entryName
}

func (ag *ActionGraph) GetEntry() (NodeEntryInterface, error) {
	node, exists := ag.Nodes[ag.Entry]
	if !exists {
		return nil, fmt.Errorf("entry %s not found", ag.Entry)
	}

	execNode, ok := node.(NodeEntryInterface)
	if !ok {
		return nil, fmt.Errorf("entry %s is not a start node", ag.Entry)
	}

	return execNode, nil
}

func NewActionGraph() ActionGraph {
	return ActionGraph{
		Nodes: make(map[string]NodeRef),
	}
}

func loadEntry(ag *ActionGraph, nodesYaml map[string]interface{}) error {
	entryAny, exists := nodesYaml["entry"]
	if !exists {
		return fmt.Errorf("entry is missing")
	}

	entry, ok := entryAny.(string)
	if !ok {
		return fmt.Errorf("entry is not a string")
	}

	ag.SetEntry(entry)
	return nil
}

func RunGraph(graphContent []byte) error {
	ag, err := LoadGraph(graphContent)
	if err != nil {
		return err
	}

	entry, err := ag.GetEntry()
	if err != nil {
		return err
	}

	err = entry.ExecuteEntry(nil)
	if err != nil {
		return err
	}
	return nil
}

func LoadGraph(graphContent []byte) (ActionGraph, error) {

	ag := NewActionGraph()

	graphYaml := make(map[string]any)
	err := yaml.Unmarshal(graphContent, &graphYaml)
	if err != nil {
		return ActionGraph{}, u.Throw(err)
	}

	// Load Nodes
	err = loadNodes(&ag, graphYaml)
	if err != nil {
		return ActionGraph{}, u.Throw(err)
	}

	// Load Executions
	err = loadExecutions(&ag, graphYaml)
	if err != nil {
		return ActionGraph{}, u.Throw(err)
	}

	// Load connections
	err = loadConnections(&ag, graphYaml)
	if err != nil {
		return ActionGraph{}, u.Throw(err)
	}

	// Load Entry
	err = loadEntry(&ag, graphYaml)
	if err != nil {
		return ActionGraph{}, u.Throw(err)
	}

	// load group inputs and outputs if available
	inputs, ok := graphYaml["inputs"]
	if ok {
		idefs := make(map[InputId]InputDefinition)
		for k, v := range inputs.(map[string]any) {
			idef, err := anyToPortDefinition[InputDefinition](v)
			if err != nil {
				return ActionGraph{}, err
			}

			idefs[InputId(k)] = idef
		}
		ag.Inputs = idefs
	}

	outputs, ok := graphYaml["outputs"]
	if ok {
		odefs := make(map[OutputId]OutputDefinition)
		for k, v := range outputs.(map[string]any) {
			odef, err := anyToPortDefinition[OutputDefinition](v)
			if err != nil {
				return ActionGraph{}, err
			}

			odefs[OutputId(k)] = odef
		}
		ag.Outputs = odefs
	}

	return ag, nil
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

func loadNodes(ag *ActionGraph, nodesYaml map[string]interface{}) error {
	nodesList, err := u.GetItem[[]any](nodesYaml, "nodes")
	if err != nil {
		return u.Throw(err)
	}

	for _, node := range nodesList {
		nodeI, ok := node.(map[string]any)
		if !ok {
			return fmt.Errorf("node is not a map")
		}

		id, err := u.GetItem[string](nodeI, "id")
		if err != nil {
			return u.Throw(err)
		}

		nodeType, err := u.GetItem[string](nodeI, "type")
		if err != nil {
			return u.Throw(err)
		}

		var node NodeRef

		if strings.HasPrefix(nodeType, "github.com/") {
			node, err = NewGhActionNode(nodeType)
		} else {
			node, err = NewNodeInstance(nodeType, nodeI)
		}
		if err != nil {
			return u.Throw(err)
		}

		// If there are user input values, then set them to the input values array
		_, exists := nodeI["inputs"]
		if exists {
			// If node has inputs defined in yaml, set them
			inputs, hasInputs := node.(HasInputsInterface)
			if hasInputs {
				is, err := u.GetItem[map[string]any](nodeI, "inputs")
				if err != nil {
					return u.Throw(err)
				}

				for key, value := range is {
					err = inputs.SetInputValue(InputId(key), value)
					if err != nil {
						return u.Throw(err)
					}
				}
			}
		}

		node.SetId(id)
		ag.AddNode(id, node)
	}
	return nil
}

func loadExecutions(ag *ActionGraph, nodesYaml map[string]interface{}) error {

	executionList, err := u.GetItem[[]interface{}](nodesYaml, "executions")
	if err != nil {
		return u.Throw(err)
	}

	for _, execution := range executionList {
		e, ok := execution.(map[string]interface{})
		if !ok {
			return fmt.Errorf("execution is not a map")
		}

		srcNodeId, err := u.GetItem[string](e, "src", "node")
		if err != nil {
			return u.Throw(err)
		}

		srcNodePortId, err := u.GetItem[string](e, "src", "port")
		if err != nil {
			return u.Throw(err)
		}

		dstNodeId, err := u.GetItem[string](e, "dst", "node")
		if err != nil {
			return u.Throw(err)
		}

		dstNodePortId, err := u.GetItem[string](e, "dst", "port")
		if err != nil {
			return u.Throw(err)
		}

		dstNode, err := ag.FindNode(dstNodeId)
		if err != nil {
			return fmt.Errorf("execution dst node does not exist")
		}

		srcNode, err := ag.FindNode(srcNodeId)
		if err != nil {
			return fmt.Errorf("execution src node does not exist")
		}

		if strings.HasPrefix(dstNode.GetNodeType(), "group@") {
			subgraph := dstNode.GetGraph()
			if subgraph == nil {
				return fmt.Errorf("group node has no sub graph")
			}

			groupStart, err := subgraph.FindNode(subgraph.Entry)
			if err != nil {
				return fmt.Errorf("sub graph has no entry")
			}

			dstNode = groupStart
			// 'exec' stays the same
		} else if strings.HasPrefix(srcNode.GetNodeType(), "group@") {
			subgraph := srcNode.GetGraph()
			if subgraph == nil {
				return fmt.Errorf("group node has no sub graph")
			}
			var groupEnd NodeRef
			for _, node := range subgraph.GetNodes() {
				if strings.HasPrefix(node.GetNodeType(), "group-output@") {
					groupEnd = node
					break
				}
			}
			if groupEnd == nil {
				return fmt.Errorf("group has no output")
			}

			srcNode = groupEnd
			// 'exec' stays the same
		}

		vSrcNode := reflect.ValueOf(srcNode).Elem()
		if !vSrcNode.IsValid() {
			return fmt.Errorf("executions src node is not valid")
		}

		v := reflect.ValueOf(srcNode)
		ConnectExecutionPort := v.MethodByName("ConnectExecutionPort")

		args := []reflect.Value{reflect.ValueOf(
			OutputId(OutputId(srcNodePortId))),
			reflect.ValueOf(dstNode),
			reflect.ValueOf(InputId(dstNodePortId)),
		}
		ConnectExecutionPort.Call(args)
	}

	return nil
}

func loadConnections(ag *ActionGraph, nodesYaml map[string]any) error {

	connectionsList, err := u.GetItem[[]interface{}](nodesYaml, "connections")
	if err != nil {
		return u.Throw(err)
	}

	for _, connection := range connectionsList {
		c, ok := connection.(map[string]interface{})
		if !ok {
			return fmt.Errorf("connection is not a map")
		}

		srcNodeId, err := u.GetItem[string](c, "src", "node")
		if err != nil {
			return u.Throw(err)
		}

		dstNodeId, err := u.GetItem[string](c, "dst", "node")
		if err != nil {
			return u.Throw(err)
		}

		srcPort, err := u.GetItem[string](c, "src", "port")
		if err != nil {
			return u.Throw(err)
		}

		dstPort, err := u.GetItem[string](c, "dst", "port")
		if err != nil {
			return u.Throw(err)
		}

		srcNode, err := ag.FindNode(srcNodeId)
		if err != nil {
			return fmt.Errorf("connection src node does not exist")
		}

		dstNode, err := ag.FindNode(dstNodeId)
		if err != nil {
			return fmt.Errorf("connection dst node does not exist")
		}

		v := reflect.ValueOf(dstNode)
		ConnectDataPort := v.MethodByName("ConnectDataPort")

		source := reflect.ValueOf(DataSource{
			Output:  OutputId(srcPort),
			SrcNode: srcNode.(HasOutputsInterface),
		})

		ConnectDataPort.Call([]reflect.Value{reflect.ValueOf(InputId(dstPort)), source})
	}
	return nil
}
