package core

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

type InputId string
type OutputId string

type NodeRef NodeBaseInterface

// An interface for nodes that execute their logic.
type NodeExecutionInterface interface {
	ExecuteImpl(c ExecutionContext) error
	GetNodeType() string
	GetId() string
}

// An interface for nodes that can kick off an action graph.
type NodeEntryInterface interface {
	ExecuteEntry() error
}

type NodeBaseInterface interface {
	SetNodeType(name string)
	SetId(id string)
	GetNodeType() string
	GetId() string
}

// Base component for nodes that offer values from other nodes.
// The node that implements this component has outgoing connections.
type NodeBaseComponent struct {
	Id       string // Unique identifier for the node
	nodeType string // Name of the node
}

func (n *NodeBaseComponent) SetId(id string) {
	n.Id = id
}

func (n *NodeBaseComponent) GetId() string {
	return n.Id
}

func (n *NodeBaseComponent) GetNodeType() string {
	return n.nodeType
}

func (n *NodeBaseComponent) SetNodeType(name string) {
	n.nodeType = name
}

func (n *NodeBaseComponent) Execute(t NodeExecutionInterface, ec ExecutionContext) error {
	// nothing to execute
	if t == nil {
		return nil
	}

	// GitHub Action Node does its own logging
	if t.GetNodeType() != "gh-action@v1" {
		utils.LoggerBase.Printf("Execute '%s (%s)'\n",
			t.GetId(),
			t.GetNodeType(),
		)
	}

	err := t.ExecuteImpl(ec)
	if err != nil {
		return err
	}
	return nil
}

type SourceNode struct {
	Src  HasOuputsInterface
	Name OutputId
}

type nodeFactoryFunc func(context interface{}) (NodeRef, error)

type InputsOutputs struct {
	Inputs  any
	Outputs any
}

var registries = make(map[string]NodeTypeDefinitionFull)

func GetRegistries() map[string]NodeTypeDefinitionFull {
	return registries
}

type InputOption struct {
	Name  string `yaml:"name" json:"name" bson:"name"`
	Value string `yaml:"value" json:"value" bson:"value"`
}

type OutputDefinition struct {
	Name string `yaml:"name" json:"name" bson:"name"`
	Type string `yaml:"type" json:"type" bson:"type"`

	Index int `yaml:"index" json:"index" bson:"index"`

	Group        bool `yaml:"group,omitempty" json:"group,omitempty" bson:"group,omitempty"`
	GroupInitial int  `yaml:"group_initial,omitempty" json:"group_initial,omitempty" bson:"group_initial,omitempty"`

	Exec bool `yaml:"exec,omitempty" json:"exec,omitempty" bson:"exec,omitempty"`

	Description string `yaml:"description" json:"description" bson:"description"`
	Default     any    `yaml:"default,omitempty" json:"default,omitempty" bson:"default,omitempty"`
}

type InputDefinition struct {
	Type  string `yaml:"type" json:"type" bson:"type"`
	Index int    `yaml:"index" json:"index" bson:"index"`

	Name string `yaml:"name,omitempty" json:"name,omitempty" bson:"name,omitempty"`

	Group        bool `yaml:"group,omitempty" json:"group,omitempty" bson:"group,omitempty"`
	GroupInitial int  `yaml:"group_initial,omitempty" json:"group_initial,omitempty" bson:"group_initial,omitempty"`

	Exec bool `yaml:"exec,omitempty" json:"exec,omitempty" bson:"exec,omitempty"`

	Description string        `yaml:"description" json:"description" bson:"description"`
	Default     any           `yaml:"default,omitempty" json:"default,omitempty" bson:"default,omitempty"`
	Required    bool          `yaml:"required,omitempty" json:"required,omitempty" bson:"required,omitempty"`
	Options     []InputOption `yaml:"options,omitempty" json:"options,omitempty" bson:"options,omitempty"`

	// for type "string"
	Multiline bool   `yaml:"multiline,omitempty" json:"multiline,omitempty" bson:"multiline,omitempty"`
	Hint      string `yaml:"hint,omitempty" json:"hint,omitempty" bson:"hint,omitempty"`

	// for type "number"
	Step float64 `yaml:"step,omitempty" json:"step,omitempty" bson:"step,omitempty"`
}

type NodeTypeDefinitionBasic struct {
	Id          string `yaml:"id" json:"id" bson:"_id"`
	Name        string `yaml:"name" json:"name" bson:"name"`
	Version     string `yaml:"version" json:"version" bson:"version"`
	Description string `yaml:"description" json:"description" bson:"description"`
	Entry       bool   `yaml:"entry" json:"entry" bson:"entry"`
	Compact     bool   `yaml:"compact,omitempty" json:"compact,omitempty" bson:"compact,omitempty"`
	Icon        string `yaml:"icon" json:"icon" bson:"icon"`
	Avatar      string `yaml:"avatar" json:"avatar" bson:"avatar"`
	Registry    string `yaml:"registry,omitempty" json:"registry,omitempty" bson:"registry,omitempty"`
}

type NodeTypeDefinitionFull struct {
	NodeTypeDefinitionBasic `yaml:",inline" json:",inline" bson:",inline"`
	Inputs                  map[InputId]InputDefinition   `yaml:"inputs" json:"inputs" bson:"inputs"`
	Outputs                 map[OutputId]OutputDefinition `yaml:"outputs" json:"outputs" bson:"outputs"`

	Style struct {
		Header struct {
			Background string `yaml:"background" json:"background" bson:"background"`
		} `yaml:"header" json:"header" bson:"header"`
		Body struct {
			Background string `yaml:"background" json:"background" bson:"background"`
		} `yaml:"body" json:"body" bson:"body"`
	} `yaml:"style" json:"style" bson:"style"`

	// Factory function for creating a new node instance
	// Not part of the yaml definition
	FactoryFn nodeFactoryFunc `yaml:"-" json:"-" bson:"-"`
}

func RegisterNodeFactory(nodeDefinition string, fn nodeFactoryFunc) error {

	var def NodeTypeDefinitionFull
	err := yaml.Unmarshal([]byte(nodeDefinition), &def)
	if err != nil {
		return err
	}

	outputIndexes := make(map[int]string)
	inputIndexes := make(map[int]string)

	// Increase the gap between the input ports
	// to make space for sub ports
	for inputId, input := range def.Inputs {
		tmp := def.Inputs[inputId]
		prev, exists := inputIndexes[input.Index]
		if exists {
			return fmt.Errorf("duplicate input index in %v at '%v' / '%v'", def.Name, inputId, prev)
		}
		inputIndexes[input.Index] = string(inputId)

		tmp.Index = input.Index * 128 // 128 means there is space for 127 sub ports available
		def.Inputs[inputId] = tmp
	}

	// Increase the gap between the output ports
	// to make space for sub ports
	for outputId, output := range def.Outputs {
		tmp := def.Outputs[outputId]
		prev, exists := outputIndexes[output.Index]
		if exists {
			return fmt.Errorf("duplicate output index in %v at '%v' / '%v'", def.Name, outputId, prev)
		}
		outputIndexes[output.Index] = string(outputId)

		tmp.Index = output.Index * 128 // 128 means there is space for 127 sub ports available
		def.Outputs[outputId] = tmp
	}

	id := fmt.Sprintf("%v@v%v", def.Id, def.Version)
	_, ok := registries[id]
	if ok {
		return fmt.Errorf("node definition '%v' already registered", nodeDefinition)
	}

	def.FactoryFn = fn
	registries[id] = def

	return nil
}

func NewGhActionNode(nodeType string) (NodeRef, error) {
	factoryEntry, exists := registries["gh-action@v1"]
	if !exists {
		return nil, fmt.Errorf("node type '%v' not registered", nodeType)
	}

	node, err := factoryEntry.FactoryFn(nodeType)
	if err != nil {
		return nil, err
	}

	utils.InitMapAndSliceInStructRecursively(reflect.ValueOf(node))
	return node, nil
}

func NewNodeInstance(nodeType string) (NodeRef, error) {
	var (
		node NodeRef
		err  error
	)
	factoryEntry, exists := registries[nodeType]
	if exists {
		node, err = factoryEntry.FactoryFn(nil)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown node type '%v'", nodeType)
	}

	inputNode, ok := node.(HasInputsInterface)
	if ok {
		inputNode.SetInputDefs(factoryEntry.Inputs)
	}

	outputNode, ok := node.(HasOuputsInterface)
	if ok {
		outputNode.SetOutputDefs(factoryEntry.Outputs)
	}

	// Ensure that the factory function returned a pointer
	if reflect.TypeOf(node).Kind() != reflect.Ptr {
		return nil, fmt.Errorf("factory function for '%v' must return a pointer, did you forget '&' in front of the return type?", nodeType)
	}

	utils.InitMapAndSliceInStructRecursively(reflect.ValueOf(node))

	node.SetNodeType(nodeType)
	return node, nil
}
