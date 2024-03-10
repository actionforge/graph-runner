package core

import (
	"actionforge/graph-runner/utils"
	"fmt"
	"os"
	"reflect"
	"sync"
)

// HasOutputsInterface is a representation for all outputs of a node.
// The node that implements this interface has outgoing connections.
type HasOutputsInterface interface {
	OutputDefsCopy() map[OutputId]OutputDefinition
	SetOutputDefs(outputs map[OutputId]OutputDefinition)

	OutputValueById(c ExecutionContext, outputId OutputId) (value interface{}, err error)
	SetOutputValue(c ExecutionContext, outputId OutputId, value interface{}) error
	IncrementConnectionCounter(outputId OutputId)
}

type Executions struct {
	Executions  map[OutputId]NodeExecutionInterface
	PortMapping map[OutputId]InputId
}

func (n *Executions) Execute(t NodeExecutionInterface, ec ExecutionContext) error {
	// nothing to execute
	if t == nil {
		return nil
	}

	if os.Getenv("GITHUB_EVENT_NAME") != "" {
		utils.LoggerBase.Printf("🟢 Execute '%s (%s)'\n",
			t.GetName(),
			t.GetId(),
		)
	}

	err := t.ExecuteImpl(ec)
	if err != nil {
		return err
	}
	return nil
}

func (e *Executions) GetAllExecutionPorts() map[OutputId]NodeExecutionInterface {
	return e.Executions
}

func (e *Executions) GetTargetNode(portName OutputId) NodeExecutionInterface {
	return e.Executions[portName]
}

func (e *Executions) GetTargetPort(portName OutputId) InputId {
	return e.PortMapping[portName]
}

func (e *Executions) ConnectExecutionPort(portName OutputId, target NodeExecutionInterface, targetPortId InputId) {
	if e.Executions == nil {
		e.Executions = make(map[OutputId]NodeExecutionInterface)
	}
	if e.PortMapping == nil {
		e.PortMapping = make(map[OutputId]InputId)
	}

	e.PortMapping[portName] = targetPortId
	e.Executions[portName] = target
}

type Outputs struct {
	outputLock sync.RWMutex

	outputDefs   map[OutputId]OutputDefinition
	outputValues map[contextKey]map[OutputId]interface{}

	connectionCounter map[OutputId]int64
}

func (n *Outputs) IncrementConnectionCounter(outputId OutputId) {
	n.outputLock.Lock()
	defer n.outputLock.Unlock()

	if n.connectionCounter == nil {
		n.connectionCounter = make(map[OutputId]int64)
	}
	n.connectionCounter[outputId]++
}

func (n *Outputs) OutputDefsCopy() map[OutputId]OutputDefinition {
	n.outputLock.RLock()
	defer n.outputLock.RUnlock()

	outputDefsCopy := make(map[OutputId]OutputDefinition)
	for k, v := range n.outputDefs {
		outputDefsCopy[k] = v
	}
	return outputDefsCopy
}

func (n *Outputs) SetOutputDefs(outputs map[OutputId]OutputDefinition) {
	n.outputLock.RLock()
	defer n.outputLock.RUnlock()

	n.outputDefs = outputs
}

func (n *Outputs) OutputValueById(c ExecutionContext, outputId OutputId) (interface{}, error) {

	n.outputLock.RLock()
	defer n.outputLock.RUnlock()

	_, outputExists := n.outputDefs[outputId]
	if !outputExists {
		return nil, fmt.Errorf("output '%v' doesn't exist", outputId)
	}

	for _, ck := range c.GetContextKeys(nil) {
		threadValuePool, exists := n.outputValues[ck]
		if exists {
			outputValue, exists := threadValuePool[outputId]
			if exists {
				return outputValue, nil
			}
		}
	}

	return nil, fmt.Errorf("no value for output '%v'", outputId)
}

// SetOutputValue sets the value of an output to the node.
// The value type must match the output type, otherwise an error
// is returned.
func (n *Outputs) SetOutputValue(c ExecutionContext, outputId OutputId, value interface{}) error {

	// if the output is not connected, we don't need to keep the value alive
	connectionCounter := n.connectionCounter[outputId]
	if connectionCounter == 0 {
		return nil
	}

	n.outputLock.Lock()
	defer n.outputLock.Unlock()

	output, outputExists := n.outputDefs[outputId]
	if !outputExists {

		// if the output could not be found,
		// check if it is a sub port instead
		sb := getSubPortRegex().FindStringSubmatch(string(outputId))
		if len(sb) < 2 {
			return fmt.Errorf("unknown output '%v'", outputId)
		}

		output, outputExists = n.outputDefs[OutputId(sb[1])]
		if !outputExists {
			// If still nothing found, return an error
			return fmt.Errorf("unknown output '%v'", outputId)
		}
	}

	if !isValidOutputType(value, output.Type) {
		return fmt.Errorf("type mismatch: expected %v, got %T", output.Type, value)
	}

	if n.outputValues == nil {
		n.outputValues = make(map[contextKey]map[OutputId]interface{})
	}

	ti := c.GetLastContextKey()
	if n.outputValues[ti] == nil {
		n.outputValues[ti] = make(map[OutputId]interface{})
	}

	n.outputValues[ti][outputId] = value
	return nil
}

func isValidOutputType(value interface{}, expectedType string) bool {
	valueType := reflect.TypeOf(value)
	if valueType == nil {
		return false
	}

	switch expectedType {
	case "string":
		return valueType.Kind() == reflect.String
	case "number":
		return isNumericType(valueType)
	case "bool":
		return valueType.Kind() == reflect.Bool
	case "[]string":
		return valueType.Kind() == reflect.Slice && valueType.Elem().Kind() == reflect.String
	case "[]number":
		return valueType.Kind() == reflect.Slice && isNumericType(valueType.Elem())
	case "[]bool":
		return valueType.Kind() == reflect.Slice && valueType.Elem().Kind() == reflect.Bool
	case "any", "unknown":
		return true
	case "stream":
		return valueType.Kind() == reflect.Interface
	default:
		return valueType.String() == expectedType
	}
}

func isNumericType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return true
	default:
		return false
	}
}
