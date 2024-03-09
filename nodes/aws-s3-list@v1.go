package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"context"
	_ "embed"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:embed aws-s3-list@v1.yml
var awsS3ListDefinition string

type AwsS3ListNode struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *AwsS3ListNode) ExecuteImpl(c core.ExecutionContext) error {
	bucket, err := core.InputValueById[string](c, n.Inputs, ni.Aws_s3_list_v1_Input_bucket)
	if err != nil {
		return err
	}

	prefix, err := core.InputValueById[string](c, n.Inputs, ni.Aws_s3_list_v1_Input_prefix)
	if err != nil {
		return err
	}

	delimiter, err := core.InputValueById[string](c, n.Inputs, ni.Aws_s3_list_v1_Input_delimiter)
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return err
	}

	s3Client := s3.NewFromConfig(cfg)
	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}

	resp, err := s3Client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to list objects, %v", err)
	}

	var items []string
	for _, item := range resp.Contents {
		items = append(items, *item.Key)
	}

	err = n.Outputs.SetOutputValue(c, ni.Aws_s3_list_v1_Output_items, items)
	if err != nil {
		return err
	}

	err = n.Execute(n.GetExecutionPort(ni.Aws_s3_list_v1_Output_exec), c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(awsS3ListDefinition, func(ctx interface{}, nodeDef map[string]any) (core.NodeRef, error) {
		return &AwsS3ListNode{}, nil
	})
	if err != nil {
		panic(err)
	}
}
