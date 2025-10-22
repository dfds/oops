package aws

import (
	"context"
	"log"
	"net/http"

	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// CreateHttpClientWithoutKeepAlive Currently the AWS SDK seems to let connections live for way too long. On OSes that has a very low file descriptior limit this becomes an issue.
func CreateHttpClientWithoutKeepAlive() *awsHttp.BuildableClient {
	client := awsHttp.NewBuildableClient().WithTransportOptions(func(transport *http.Transport) {
		transport.DisableKeepAlives = true
	})

	return client
}

func AssumeRole(ctx context.Context, roleArn string) (*types.Credentials, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-west-1"), awsConfig.WithHTTPClient(CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	roleSessionName := "oops"

	assumedRole, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{RoleArn: &roleArn, RoleSessionName: &roleSessionName})
	if err != nil {
		log.Printf("unable to assume role %s, %v", roleArn, err)
		return nil, err
	}

	return assumedRole.Credentials, nil
}
