package s3

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	awsOops "go.dfds.cloud/oops/core/aws"
	"go.dfds.cloud/oops/core/config"
)

type Config struct {
	Auth   string `json:"auth"`
	Bucket string `json:"bucket"`
}

func HandleS3LocationPut(ctx context.Context, location config.BackupLocation, name string, content []byte) error {
	spec, err := config.LocationSpecToType[Config](location)
	if err != nil {
		return err
	}

	var awsCfg aws.Config
	// Determine config
	switch spec.Auth {
	case "aws-default":
		awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-west-1"), awsConfig.WithHTTPClient(awsOops.CreateHttpClientWithoutKeepAlive()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
	default:
		if spec.Auth == "" {
			awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-west-1"), awsConfig.WithHTTPClient(awsOops.CreateHttpClientWithoutKeepAlive()))
			if err != nil {
				log.Fatalf("unable to load SDK config, %v", err)
			}
		} else {
			return errors.New("unknown auth type for s3 location")
		}
	}

	backend := NewBackend(awsCfg, spec.Bucket)

	// latest
	err = backend.Put(ctx, fmt.Sprintf("latest.tar.gz"), content)
	if err != nil {
		return err
	}

	// current day
	currentTime := time.Now()
	err = backend.Put(ctx, fmt.Sprintf(fmt.Sprintf("%d/%d/%d/%d-%s", currentTime.Year(), currentTime.Month(), currentTime.Day(), currentTime.Unix(), name)), content)
	if err != nil {
		return err
	}

	return nil
}
