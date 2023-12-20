package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"runtime"
)

//go:embed switch-arch@v1.yml
var archSwitchDefinition string

type ArchSwitchNode struct {
	core.NodeBaseComponent
	core.Executions
}

func (n *ArchSwitchNode) ExecuteImpl(c core.ExecutionContext) error {

	var err error

	switch runtime.GOARCH {
	case "amd64":
		err = n.Execute(n.Executions[ni.Switch_arch_v1_Output_exec_x64], c)
	case "arm64":
		err = n.Execute(n.Executions[ni.Switch_arch_v1_Output_exec_arm64], c)
	case "arm":
		err = n.Execute(n.Executions[ni.Switch_arch_v1_Output_exec_arm32], c)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(archSwitchDefinition, func(context interface{}) (core.NodeRef, error) {
		return &ArchSwitchNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
