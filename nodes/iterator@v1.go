package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"reflect"
)

//go:embed iterator@v1.yml
var iteratorDefinition string

type IteratorNode struct {
	core.NodeBaseComponent
	core.Inputs
	core.Outputs
	core.Executions
}

func (n *IteratorNode) ExecuteImpl(ti core.ExecutionContext) error {

	iter := func(key any, value any) error {
		err := n.Outputs.SetOutputValue(ti, ni.Iterator_v1_Output_key, key)
		if err != nil {
			return err
		}

		err = n.Outputs.SetOutputValue(ti, ni.Iterator_v1_Output_value, value)
		if err != nil {
			return err
		}

		err = n.Execute(n.GetExecutionPort(ni.Iterator_v1_Output_exec), ti)
		if err != nil {
			return u.Throw(err)
		}
		return nil
	}

	iterable, err := core.InputValueById[interface{}](ti, n.Inputs, ni.Iterator_v1_Input_array)
	if err != nil {
		return err
	}

	v := reflect.ValueOf(iterable)

	switch v.Kind() {
	case reflect.String:
		for i := 0; i < v.Len(); i++ {
			ch := []rune(v.String())[i]
			err := iter(i, string(ch))
			if err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			err := iter(i, v.Index(i).Interface())
			if err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			err := iter(key, v.MapIndex(key))
			if err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("type %s is not iterable", v.Kind())
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(iteratorDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &IteratorNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
