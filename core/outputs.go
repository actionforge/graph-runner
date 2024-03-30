package core

import (
	"fmt"
	"reflect"
	"sync"
)

// HasOuputsInterface is a representation for all outputs of a node.
// The node that implements this interface has outgoing connections.
type HasOuputsInterface interface {
	OutputDefsCopy() map[OutputId]OutputDefinition
	SetOutputDefs(outputs map[OutputId]OutputDefinition)

	OutputValueById(c ExecutionContext, outputId OutputId) (value interface{}, err error)
	SetOutputValue(c ExecutionContext, outputId OutputId, value interface{}) error
}

type Executions map[OutputId]NodeExecutionInterface

type Outputs struct {
	outputLock sync.RWMutex

	outputDefs   map[OutputId]OutputDefinition
	outputValues map[string]map[OutputId]interface{}
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

	for _, ck := range c.GetContextKeysCopy(nil) {
		threadValuePool, exists := n.outputValues[ck.Id]
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

	if n.outputValues == nil {
		n.outputValues = make(map[string]map[OutputId]interface{})
	}

	if !isValidOutputType(value, output.Type) {
		return fmt.Errorf("type mismatch: expected %v, got %T", output.Type, value)
	}

	ti := c.GetLastContextKey()
	if n.outputValues[ti.Id] == nil {
		n.outputValues[ti.Id] = make(map[OutputId]interface{})
	}

	n.outputValues[ti.Id][outputId] = value
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
	case "any":
		return true
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
