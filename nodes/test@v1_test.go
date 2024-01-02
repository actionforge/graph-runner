//go:build integration_tests

package nodes

import (
	"testing"

	// initialize all nodes
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
)

// Test that the node type exists.
func Test_NewNodeInstance_Exists(t *testing.T) {
	n, err := core.NewNodeInstance("run@v1")
	if err != nil {
		t.Fatal(err)
	}

	if n.GetNodeType() != "run@v1" {
		t.Errorf("Expected node name to be 'run@v1', got '%s'", n.GetNodeType())
	}
}

// Test that the node type doesn't exist
func Test_NewNodeInstance_NotExists(t *testing.T) {
	_, err := core.NewNodeInstance("abc@v2")
	if err == nil {
		t.Error("Expected error")
		return
	}
}

// Test to ensure that all output values
// with matching types can be set and don't fail.
func Test_SetOutputValue_Success(t *testing.T) {
	node, err := core.NewNodeInstance("test@v1")
	if err != nil {
		t.Error(err)
		return
	}

	ec := core.EmptyExecutionContext()

	nodeOutputs, ok := node.(core.HasOuputsInterface)
	if !ok {
		t.Fatal("Node does not implement HasOuputsInterface")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_bool, true)
	if err != nil {
		t.Fatal(err)
	}

	// number accepts all number types, including bool
	allNumberTypes := []interface{}{
		bool(true), int(1), int8(1), int16(1), int32(1),
		int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1),
	}
	for _, v := range allNumberTypes {
		err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_number, v)
		if err != nil {
			t.Fatal("For value", v, "got error", err)
		}
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_string, "abc")
	if err != nil {
		t.Fatal(err)
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_string, []string{"foo", "bar", "bas"})
	if err != nil {
		t.Fatal(err)
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_number, []int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_bool, []bool{true, false, true})
	if err != nil {
		t.Fatal(err)
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_map_string_int32, map[string]int32{"foo": 1, "bar": 2})
	if err != nil {
		t.Fatal(err)
	}

	// ensure that any value can be set
	arr := []interface{}{1, "foo", true, map[string]int32{"foo": 1, "bar": 2}}
	for _, v := range arr {
		err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_any, v)
		if err != nil {
			t.Fatal("For value", v, "got error", err)
		}
	}
}

// Test to ensure that all output values
// with mismatching types are declined.
func Test_SetOutputValue_Decline(t *testing.T) {
	node, err := core.NewNodeInstance("test@v1")
	if err != nil {
		t.Error(err)
		return
	}

	ec := core.EmptyExecutionContext()

	nodeOutputs, ok := node.(core.HasOuputsInterface)
	if !ok {
		t.Fatal("Node does not implement HasOuputsInterface")
	}

	// nil is not a valid value for any type
	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_bool, nil)
	if err != nil {
		if err.Error() != "type mismatch: expected bool, got <nil>" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_bool, "hello")
	if err != nil {
		if err.Error() != "type mismatch: expected bool, got string" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_number, "world")
	if err != nil {
		if err.Error() != "type mismatch: expected number, got string" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_string, true)
	if err != nil {
		if err.Error() != "type mismatch: expected string, got bool" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_string, []bool{true, false, true})
	if err != nil {
		if err.Error() != "type mismatch: expected []string, got []bool" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_number, []string{"foo", "bar", "bas"})
	if err != nil {
		if err.Error() != "type mismatch: expected []number, got []string" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_bool, []int{1, 2, 3})
	if err != nil {
		if err.Error() != "type mismatch: expected []bool, got []int" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}

	err = nodeOutputs.SetOutputValue(ec, ni.Test_v1_Output_output_map_string_int32, map[string]string{"foo": "bar"})
	if err != nil {
		if err.Error() != "type mismatch: expected map[string]int32, got map[string]string" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("Expected error")
	}
}

// Test that connects two nodes and requests the output value from the first one.
// Used to ensure requesting
func Test_InputValueById_Match(t *testing.T) {

	test1Node, test1Outputs, test2Node, test2Inputs := createTwoNodesAndConnectThem(t)

	ec := core.EmptyExecutionContext()

	// Connect all ports
	test2Inputs.ConnectPort(ni.Test_v1_Input_input_string, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_string,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_number, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_number,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_bool, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_bool,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_string, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_string,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_number, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_number,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_bool, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_bool,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_map_string_int32, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_map_string_int32,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_any, core.SourceNode{
		Src:  test1Node.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_any,
	})

	// bool

	err := test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_bool, true)
	if err != nil {
		t.Fatal(err)
	}

	b, err := core.InputValueById[bool](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_bool)
	if err != nil {
		t.Fatal(err)
	}

	if !b {
		t.Error("Expected input value to be 'true', got", b)
	}

	a, err := core.InputValueById[interface{}](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_bool)
	if err != nil {
		t.Fatal(err)
	}

	if a != true {
		t.Error("Expected input value to be 'true', got", a)
	}

	// number

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_number, 123)
	if err != nil {
		t.Fatal(err)
	}

	n, err := core.InputValueById[int](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_number)
	if err != nil {
		t.Fatal(err)
	}

	if n != 123 {
		t.Error("Expected input value to be '123', got", n)
	}

	// string

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_string, "abc")
	if err != nil {
		t.Fatal(err)
	}

	s, err := core.InputValueById[string](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_string)
	if err != nil {
		t.Fatal(err)
	}

	if s != "abc" {
		t.Error("Expected input value to be 'abc', got", s)
	}

	// array string

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_string, []string{"foo", "bar", "bas"})
	if err != nil {
		t.Fatal(err)
	}

	arrStr, err := core.InputValueById[[]string](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_string)
	if err != nil {
		t.Fatal(err)
	}

	if len(arrStr) != 3 {
		t.Error("Expected input value to be of length 3, got", len(arrStr))
	}

	if arrStr[0] != "foo" && arrStr[1] != "bar" && arrStr[2] != "bas" {
		t.Error("Expected input value to be 'foo,bar,bas', got", arrStr)
	}

	// array number

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_number, []int{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}

	arrNum, err := core.InputValueById[[]int](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_number)
	if err != nil {
		t.Fatal(err)
	}

	if len(arrNum) != 3 {
		t.Error("Expected input value to be of length 3, got", len(arrNum))
	}

	if arrNum[0] != 1 && arrNum[1] != 2 && arrNum[2] != 3 {
		t.Error("Expected input value to be '1,2,3', got", arrNum)
	}

	// array bool

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_bool, []bool{true, false, true})
	if err != nil {
		t.Fatal(err)
	}

	arrBool, err := core.InputValueById[[]bool](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_bool)
	if err != nil {
		t.Fatal(err)
	}

	if len(arrBool) != 3 {
		t.Error("Expected input value to be of length 3, got", len(arrBool))
	}

	if arrBool[0] != true && arrBool[1] != false && arrBool[2] != true {
		t.Error("Expected input value to be 'true,false,true', got", arrBool)
	}

	// map string int32

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_map_string_int32, map[string]int32{"foo": 1, "bar": 2})
	if err != nil {
		t.Fatal(err)
	}

	m, err := core.InputValueById[map[string]int32](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_map_string_int32)
	if err != nil {
		t.Fatal(err)
	}

	if len(m) != 2 {
		t.Error("Expected input value to be of length 2, got", len(m))
	}

	if m["foo"] != 1 && m["bar"] != 2 {
		t.Error("Expected input value to be 'foo:1,bar:2', got", m)
	}

	// any

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_any, "abc")
	if err != nil {
		t.Fatal(err)
	}

	any, err := core.InputValueById[any](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_any)
	if err != nil {
		t.Fatal(err)
	}

	if any != "abc" {
		t.Error("Expected input value to be 'abc', got", any)
	}
}

// Test that connects two nodes and requests the output value from the first one.
// The value of the output value is correct, but the requested type requires an implicit cast.
// E.g:
//   - output value is 'true' and the input value is bool.
//   - the requested type is int32, so an implicit cast from bool to int32 must occur.
func Test_InputValueById_Casting(t *testing.T) {

	_, test1Outputs, test2Node, _ := createTwoNodesAndConnectThem(t)

	ec := core.EmptyExecutionContext()

	// The following outputs are not tested as they cannot be casted to other types:
	// - Test_v1_Output_output_string
	// - Test_v1_Output_output_array_string
	// - Test_v1_Output_output_map_string_int32

	// test bool to int32

	err := test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_bool, true)
	if err != nil {
		t.Fatal(err)
	}

	b, err := core.InputValueById[int32](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_bool)
	if err != nil {
		t.Fatal(err)
	}

	if b == 0 {
		t.Error("Expected input value to be 'true', got", b)
	}

	// test int8 to bool and int32

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_number, int8(42))
	if err != nil {
		t.Fatal(err)
	}

	n1, err := core.InputValueById[bool](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_number)
	if err != nil {
		t.Fatal(err)
	}

	if n1 != true {
		t.Error("Expected input value to be 'true', got", n1)
	}

	n2, err := core.InputValueById[int32](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_number)
	if err != nil {
		t.Fatal(err)
	}

	if n2 != 42 {
		t.Error("Expected input value to be '42', got", n2)
	}

	// array int to array bool

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_number, []int{0, 1, 4})
	if err != nil {
		t.Fatal(err)
	}

	a1, err := core.InputValueById[[]bool](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_number)
	if err != nil {
		t.Fatal(err)
	}

	if len(a1) != 3 {
		t.Error("Expected input value to be of length 3, got", len(a1))
	}

	if a1[0] != false && a1[1] != true && a1[2] != true {
		t.Error("Expected input value to be 'false,true,true', got", a1)
	}

	a2, err := core.InputValueById[[]int16](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_number)
	if err != nil {
		t.Fatal(err)
	}

	if len(a2) != 3 {
		t.Error("Expected input value to be of length 3, got", len(a2))
	}

	if a2[0] != 0 && a2[1] != 1 && a2[2] != 4 {
		t.Error("Expected input value to be '0,1,4', got", a2)
	}

	// array int to array int32

	err = test1Outputs.SetOutputValue(ec, ni.Test_v1_Output_output_array_bool, []bool{true, false, true})
	if err != nil {
		t.Fatal(err)
	}

	a3, err := core.InputValueById[[]int32](ec, test2Node.(*TestNode).Inputs, ni.Test_v1_Input_input_array_bool)
	if err != nil {
		t.Fatal(err)
	}

	if len(a3) != 3 {
		t.Error("Expected input value to be of length 3, got", len(a3))
	}

	if a3[0] != 1 && a3[1] != 0 && a3[2] != 1 {
		t.Error("Expected input value to be '1,0,1', got", a3)
	}
}

func createTwoNodesAndConnectThem(t *testing.T) (src core.NodeRef, srcOutputs core.HasOuputsInterface, dst core.NodeRef, dstInputs core.HasInputsInterface) {
	src, err := core.NewNodeInstance("test@v1")
	if err != nil {
		t.Fatal(err)
	}

	dst, err = core.NewNodeInstance("test@v1")
	if err != nil {
		t.Fatal(err)
	}

	testOutputs, ok := src.(core.HasOuputsInterface)
	if !ok {
		t.Fatal("Node does not implement HasOuputsInterface")
	}

	test2Inputs, ok := dst.(core.HasInputsInterface)
	if !ok {
		t.Fatal("Node does not implement HasInputsInterface")
	}

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_string, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_string,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_number, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_number,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_bool, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_bool,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_string, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_string,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_number, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_number,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_array_bool, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_array_bool,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_map_string_int32, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_map_string_int32,
	})

	test2Inputs.ConnectPort(ni.Test_v1_Input_input_any, core.SourceNode{
		Src:  src.(core.HasOuputsInterface),
		Name: ni.Test_v1_Output_output_any,
	})

	return src, testOutputs, dst, test2Inputs
}
