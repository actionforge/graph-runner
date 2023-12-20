package main

import (
	"actionforge/graph-runner/cmd"
	_ "actionforge/graph-runner/cmd"
	_ "actionforge/graph-runner/unit_tests"
	"actionforge/graph-runner/utils"
	"fmt"
)

func main() {
	// License info
	// This code must not be removed or bypassed
	fmt.Println("Actionforge Graph Runner (non-commercial)")

	features := utils.GetFeatureString()
	if len(features) > 0 {
		fmt.Println("Features enabled: " + features)
	}

	cmd.Execute()
}
