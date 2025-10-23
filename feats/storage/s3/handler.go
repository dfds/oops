package s3

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsOops "go.dfds.cloud/oops/core/aws"
	"go.dfds.cloud/oops/core/config"
	"go.dfds.cloud/oops/core/logging"
	"go.uber.org/zap"
)

type Config struct {
	Auth    string `json:"auth"`
	Bucket  string `json:"bucket"`
	RoleArn string `json:"roleArn"`
	Region  string `json:"region"`
}

func HandleS3LocationPut(ctx context.Context, location config.BackupLocation, name string, content []byte) error {
	spec, err := config.LocationSpecToType[Config](location)
	if err != nil {
		return err
	}

	var awsCfg aws.Config
	// Determine config
	switch spec.Auth {
	case "aws-assume":
		creds, err := awsOops.AssumeRole(ctx, spec.RoleArn)
		if err != nil {
			return err
		}
		awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)), awsConfig.WithRegion(spec.Region))
		if err != nil {
			return err
		}
	case "aws-default":
		awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(spec.Region), awsConfig.WithHTTPClient(awsOops.CreateHttpClientWithoutKeepAlive()))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
	default:
		if spec.Auth == "" {
			awsCfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(spec.Region), awsConfig.WithHTTPClient(awsOops.CreateHttpClientWithoutKeepAlive()))
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

	logging.Logger.Info("Saved backup to storage location", zap.String("location", location.Name), zap.String("provider", location.Provider), zap.String("bucket", spec.Bucket))

	return nil
}
