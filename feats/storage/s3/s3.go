package s3

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Backend struct {
	client *s3.Client
	bucket string
}

func NewBackend(awsConf aws.Config, bucket string) *Backend {
	return &Backend{
		client: s3.NewFromConfig(awsConf),
		bucket: bucket,
	}
}

func (s *Backend) Put(ctx context.Context, path string, content []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Backend) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Backend) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &path,
	})
	if err != nil {
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
