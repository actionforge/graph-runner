package nodes

import (
	"actionforge/graph-runner/core"
	ni "actionforge/graph-runner/node_interfaces"
	"actionforge/graph-runner/utils"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

//go:embed aws-s3-upload@v1.yml
var awsS3Definition string

type AwsS3Node struct {
	core.NodeBaseComponent
	core.Executions
	core.Inputs
	core.Outputs
}

func (n *AwsS3Node) ExecuteImpl(c core.ExecutionContext) error {
	input, err := core.InputValueById[any](c, n.Inputs, ni.Aws_s3_upload_v1_Input_content)
	if err != nil {
		return err
	}

	name, err := core.InputValueById[string](c, n.Inputs, ni.Aws_s3_upload_v1_Input_name)
	if err != nil {
		return err
	}

	bucket, err := core.InputValueById[string](c, n.Inputs, ni.Aws_s3_upload_v1_Input_bucket)
	if err != nil {
		return err
	}

	config, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	fmt.Println("Upload to bucket: ", bucket, " with name: ", name)

	bb := BucketBasics{
		S3Client: s3.NewFromConfig(config),
	}

	reader, err := utils.AnyToReader(input)
	if err != nil {
		return err
	}

	cleanup := func() {
		if f := reader.(*os.File); f != nil {
			_ = f.Close()
		}
	}

	err = bb.UploadFile(bucket, name, reader)

	cleanup()

	if err != nil {
		return err
	}

	err = n.Execute(n.Executions[ni.Aws_s3_upload_v1_Output_exec], c)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	err := core.RegisterNodeFactory(awsS3Definition, func(context interface{}) (core.NodeRef, error) {
		return &AwsS3Node{}, nil
	})
	if err != nil {
		panic(err)
	}
}

type BucketBasics struct {
	S3Client *s3.Client
}

func (basics BucketBasics) UploadFile(bucketName string, objectKey string, object io.Reader) error {
	_, err := basics.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   object,
	})
	if err != nil {
		return fmt.Errorf("failed to upload object, %v", err)
	}
	return nil
}
