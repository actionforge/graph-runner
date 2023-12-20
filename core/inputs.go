package core

import (
	u "actionforge/graph-runner/utils"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/exp/maps"
)

var (
	onceSubPortRegex sync.Once
	subPortRegex     *regexp.Regexp
)

var defaultZeroValues = map[string]interface{}{
	"string":   "",
	"number":   0,
	"bool":     false,
	"any":      nil,
	"[]number": []float64{},
	"[]bool":   []bool{},
	"[]string": []string{},
	"[]any":    []interface{}{},
}

// HasInputsInterface is a representation for all inputs of a node.
// The node that implements this interface has incoming connections.
type HasInputsInterface interface {
	InputDefsCopy() map[InputId]InputDefinition
	SetInputDefs(inputs map[InputId]InputDefinition)

	InputValueById(c ExecutionContext, inputId InputId, group *InputId) (value interface{}, err error)
	SetInputValue(inputId InputId, value interface{}) error

	ConnectPort(dstname InputId, src SourceNode)
}

type Inputs struct {
	inputDefs   map[InputId]InputDefinition
	inputValues map[InputId]interface{}

	incomingNodes map[InputId]HasOuputsInterface

	// Map which input is connected to which output
	inputToOutputMapping map[InputId]OutputId
}

func getSubPortRegex() *regexp.Regexp {
	onceSubPortRegex.Do(func() {
		subPortRegex = regexp.MustCompile(`^([\w]+)\[([0-9]+)\]$`)
	})
	return subPortRegex
}

func (n *Inputs) InputDefsCopy() map[InputId]InputDefinition {

	inputDefsCopy := make(map[InputId]InputDefinition)
	for k, v := range n.inputDefs {
		inputDefsCopy[k] = v
	}
	return inputDefsCopy
}

func (n *Inputs) ConnectPort(dstname InputId, src SourceNode) {
	if n.incomingNodes == nil {
		n.incomingNodes = make(map[InputId]HasOuputsInterface)
	}
	if n.inputToOutputMapping == nil {
		n.inputToOutputMapping = make(map[InputId]OutputId)
	}

	n.inputToOutputMapping[dstname] = src.Name
	n.incomingNodes[dstname] = src.Src
}

func (n *Inputs) GetInputDefs() map[InputId]InputDefinition {
	return n.inputDefs
}

func (n *Inputs) SetInputDefs(inputs map[InputId]InputDefinition) {
	n.inputDefs = inputs
}

func (n *Inputs) SetInputValue(inputId InputId, value interface{}) error {
	// TODO: (Seb) Ensure that only input values are set
	// that are defined in the node definition.

	if n.inputValues == nil {
		n.inputValues = make(map[InputId]interface{})
	}

	n.inputValues[inputId] = value

	return nil
}

func (n *Inputs) GetInputValues() map[InputId]interface{} {
	return n.inputValues
}

func (n *Inputs) InputValueById(c ExecutionContext, inputId InputId, group *InputId) (value interface{}, err error) {

	// First check if there is an incoming connection...
	sourceNode, exists := n.incomingNodes[inputId]
	if exists {
		outputId, exists := n.inputToOutputMapping[inputId]
		if !exists {
			return nil, fmt.Errorf("no connection for input %v", inputId)
		}

		v, err := sourceNode.OutputValueById(c, outputId)
		if err != nil {
			return nil, u.Throw(err)
		}

		// Option inputs can be controlled by strings and number outputs.
		// Below is some additional logic in order to convert the incoming
		// value to the correct input value.
		inputDef, inputDefExists := n.inputDefs[inputId]
		if inputDefExists && inputDef.Type == "option" {
			switch c := v.(type) {
			case string:
				// Sanitize input for options coming from other nodes
				// as they might accidentally contain newlines and spaces,
				// e.g. if it comes from Python using `print("option1")`
				v = strings.Trim(c, " \n\r")
			case int8, int16, int32, int64, int, uint8, uint16, uint32, uint64, uint:
				// convert to int64
				nv := reflect.ValueOf(c).Int()
				if len(inputDef.Options) > 0 && int(nv) >= len(inputDef.Options) {
					return nil, fmt.Errorf("option value out of range: %v", nv)
				}

				v = inputDef.Options[nv].Name
			}
		}

		return v, nil
	}

	// ...If not, check if there is a user value for the input
	inputValue, exists := n.inputValues[inputId]
	if exists && inputValue != nil {
		return inputValue, nil
	}

	var (
		inputDefExists bool
		inputDef       InputDefinition
	)

	// if there is no user value, check for a default value
	if group != nil {
		// Find the group input of the sub input.
		inputDef, inputDefExists = n.inputDefs[*group]
	} else {
		inputDef, inputDefExists = n.inputDefs[inputId]
	}
	if inputDefExists {
		if inputDef.Default != nil {
			return inputDef.Default, nil
		}

		// Options are always required, but
		// also should have a default value.
		if inputDef.Type != "option" {
			// Return a zeroed value for the builtin types, including
			// any, []string, []number and []bool.
			zeroValue, exists := defaultZeroValues[inputDef.Type]
			if exists {
				return zeroValue, nil
			}

			// Last chance for foreign types. Determine if it is a slice
			// or map and at least return an empty object for these.
			if strings.HasPrefix(inputDef.Type, "[]") {
				return []interface{}{}, nil
			} else if strings.HasPrefix(inputDef.Type, "map[") {
				return map[string]interface{}{}, nil
			}
		}
	}

	return nil, &u.ErrNoInputValue{
		PortName: string(inputId),
	}
}

// InputValueById returns the value of the input with the given id.
// An error is returned if the requested type does not match
// the type of the input value.
// For sub inputs use InputValueFromSubInputs.
func InputValueById[R any](tc ExecutionContext, n Inputs, inputId InputId) (R, error) {
	return inputValueById[R](tc, n, inputId, nil)
}

// InputValueFromSubInputs returns the value of a sub input with the given id.
// An error is returned if the requested type does not match the type of the input value.
// For non sub inputs use InputValueById.
func InputValueFromSubInputs[R any](tc ExecutionContext, n Inputs, inputId InputId, group InputId) (R, error) {
	return inputValueById[R](tc, n, inputId, &group)
}

func inputValueById[R any](tc ExecutionContext, n Inputs, inputId InputId, group *InputId) (R, error) {
	var def R
	v, err := n.InputValueById(tc, inputId, group)
	if err != nil {
		return def, err
	}

	typeOfValue := reflect.TypeOf(v)
	typeOfRequested := reflect.TypeOf(def)

	// typeOfRequested/typeOfValue is nil for 'any'	type
	if typeOfRequested != nil && typeOfValue != nil {

		converted, err := convertValue(reflect.ValueOf(v), typeOfRequested)
		if err != nil {
			return def, fmt.Errorf("failed to convert value: %v", err)
		}

		v = converted.Interface()
	}

	// Final type assertion for slice case
	casted, ok := v.(R)
	if !ok {
		if typeOfValue == nil {
			return def, fmt.Errorf("value for input '%v' is not of type '%v'", inputId, typeOfRequested)
		} else {
			return def, fmt.Errorf("value for input '%v' is not of type '%v' but '%T'", inputId, typeOfRequested, typeOfValue.String())
		}
	}
	return casted, nil
}

func InputGroupValue[T any](tc ExecutionContext, n Inputs, inputId InputId) ([]T, error) {
	// Browse through all inputs of the node and collect all values
	// that belong to the requested input group.
	// An input belongs to the group if it has the same name and an index.
	// E.g: 'env[0]' where 'env' is the input name and '0' is the index.

	var def []T
	i, ok := n.GetInputDefs()[inputId]
	if !ok {
		return def, fmt.Errorf("no input definition for input '%v'", inputId)
	}

	if !i.Group {
		return def, fmt.Errorf("input '%v' is not a group input", inputId)
	}

	r := getSubPortRegex()
	sortedSubInputs := make([]string, 0)
	for _, id := range maps.Keys(n.GetInputValues()) {
		gm := r.FindStringSubmatch(string(id))
		if len(gm) > 1 && gm[1] == string(inputId) {
			sortedSubInputs = append(sortedSubInputs, gm[0])
		}
	}

	sort.Strings(sortedSubInputs)

	for _, id := range sortedSubInputs {
		v, err := InputValueFromSubInputs[T](tc, n, InputId(id), inputId)
		if err != nil {
			return def, err
		}
		def = append(def, v)
	}

	return def, nil
}

func convertValue(v reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	typeOfV := v.Type()
	if typeOfV == targetType {
		// no cast required
		return v, nil
	}
	// Unwrap interfaces and get underlying type
	if v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	switch targetType.Kind() {
	case reflect.Bool:
		return convertToBool(v)
	case reflect.String:
		return convertToString(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return convertToInt(v, targetType)
	case reflect.Float32, reflect.Float64:
		return convertToFloat(v, targetType)
	case reflect.Slice:
		if v.Kind() == reflect.Slice {
			// If the returned slice does not match the requested slice type,
			// convert each element of the slice. This is useful when the
			// returned slice is e.g. []interface{} but the requested type is []string,
			// or same if the returned slice is []int but caller requested []string.

			slice := reflect.MakeSlice(targetType, v.Len(), v.Len())
			for i := 0; i < v.Len(); i++ {
				convertedElem, err := convertValue(v.Index(i), targetType.Elem())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("failed to convert element %d: %v", i, err)
				}
				slice.Index(i).Set(convertedElem)
			}
			return slice, nil
		} else {
			return reflect.Value{}, fmt.Errorf("expected slice type but got %T", v)
		}
	}

	return reflect.Value{}, fmt.Errorf("unsupported conversion to %s", targetType)
}

func convertToString(elem reflect.Value) (reflect.Value, error) {
	switch elem.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(strconv.FormatInt(elem.Int(), 10)), nil
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(strconv.FormatFloat(elem.Float(), 'f', -1, 64)), nil
	case reflect.String:
		return elem, nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot convert %s to string", elem.Kind())
	}
}

func convertToBool(elem reflect.Value) (reflect.Value, error) {
	switch elem.Kind() {
	case reflect.String:
		boolValue, err := strconv.ParseBool(elem.String())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to convert string to bool: %v", err)
		}
		return reflect.ValueOf(boolValue), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(elem.Int() != 0), nil
	case reflect.Float32, reflect.Float64:
		return reflect.ValueOf(elem.Float() != 0), nil
	case reflect.Bool:
		return elem, nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot convert %s to bool", elem.Kind())
	}
}

func convertToInt(v reflect.Value, targetType reflect.Type) (reflect.Value, error) {

	var rv reflect.Value
	switch v.Kind() {
	case reflect.Bool:
		i := int(0)
		if v.Bool() {
			i = int(1)
		}
		rv = reflect.ValueOf(i)
	case reflect.String:
		s, err := strconv.ParseInt(v.String(), 10, targetType.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to convert string to int: %v", err)
		}
		rv = reflect.ValueOf(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i := v.Int()
		rv = reflect.ValueOf(i)
	case reflect.Float32, reflect.Float64:
		i := int64(v.Float())
		rv = reflect.ValueOf(i)
	default:
		return reflect.Value{}, fmt.Errorf("cannot convert %s to int", v.String())
	}

	if rv.CanConvert(targetType) {
		return rv.Convert(targetType), nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %s to int", v.String())
}

func convertToFloat(v reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.String:
		floatValue, err := strconv.ParseFloat(v.String(), targetType.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("failed to convert string to float: %v", err)
		}
		return reflect.ValueOf(floatValue).Convert(targetType), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflect.ValueOf(float64(v.Int())).Convert(targetType), nil
	default:
		return reflect.Value{}, fmt.Errorf("cannot convert %s to float", v.Kind())
	}
}
