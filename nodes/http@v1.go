package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	u "actionforge/graph-runner/utils"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"golang.org/x/exp/maps"
)

//go:embed http@v1.yml
var httpDefinition string

type HttpNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

var allowedMethods = map[string]bool{
	"OPTIONS": true,
	"GET":     true,
	"HEAD":    true,
	"POST":    true,
	"PUT":     true,
	"DELETE":  true,
	"TRACE":   true,
	"CONNECT": true,
}

func (n *HttpNode) ExecuteImpl(c core.ExecutionContext) error {

	method, err := core.InputValueById[string](c, n.Inputs, ni.Http_v1_Input_method)
	if err != nil {
		return err
	}

	method = strings.ToUpper(method)

	if !slices.Contains(maps.Keys(allowedMethods), method) {
		return fmt.Errorf("Invalid method: %s", method)
	}

	url, err := core.InputValueById[string](c, n.Inputs, ni.Http_v1_Input_url)
	if err != nil {
		return err
	}

	headers, err := core.InputValueById[[]string](c, n.Inputs, ni.Http_v1_Input_header)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	for _, header := range headers {
		parts := strings.SplitN(header, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("Invalid header: %s", header)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key != "" {
			req.Header.Set(key, value)
		}
	}

	req.Header.Set("User-Agent", "actionforge")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	resBody, err := io.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(c, ni.Http_v1_Output_body, string(resBody))
	if err != nil {
		return err
	}

	err = n.Outputs.SetOutputValue(c, ni.Http_v1_Output_http_status, resp.StatusCode)
	if err != nil {
		return err
	}

	exec := n.Executions[core.OutputId(ni.Http_v1_Output_exec)]
	if exec != nil {
		err = n.Execute(exec, c)
		if err != nil {
			return u.Throw(err)
		}
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(httpDefinition, func(context interface{}) (core.NodeRef, error) {
		return &HttpNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
