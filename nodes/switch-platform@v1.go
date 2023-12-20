package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"runtime"
)

//go:embed switch-platform@v1.yml
var platformSwitchDefinition string

type PlatformSwitchNode struct {
	core.NodeBaseComponent
	core.Executions
}

func (n *PlatformSwitchNode) ExecuteImpl(c core.ExecutionContext) error {

	var err error

	switch runtime.GOOS {
	case "windows":
		err = n.Execute(n.Executions[ni.Switch_platform_v1_Output_exec_win], c)
	case "linux":
		err = n.Execute(n.Executions[ni.Switch_platform_v1_Output_exec_linux], c)
	case "darwin":
		err = n.Execute(n.Executions[ni.Switch_platform_v1_Output_exec_macos], c)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		return u.Throw(err)
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(platformSwitchDefinition, func(context interface{}) (core.NodeRef, error) {
		return &PlatformSwitchNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
