// Package storages3 implements ports.StorageUploader against AWS S3 or MinIO (path-style).
package storages3

import (
	"bytes"
	"context"
	"strings"

	"weicloth/internal/core/ports"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var _ ports.StorageUploader = (*Uploader)(nil)

// Uploader uploads raw garment bytes to a single bucket.
type Uploader struct {
	client *s3.Client
	bucket string
}

// NewUploader builds an S3 client. If accessKey and secretKey are empty, uses default credential chain (e.g. IAM on EC2).
func NewUploader(ctx context.Context, region, bucket, endpointURL, accessKey, secretKey string) (*Uploader, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}
	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	ep := strings.TrimSpace(endpointURL)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if ep != "" {
			o.BaseEndpoint = aws.String(strings.TrimRight(ep, "/"))
		}
		o.UsePathStyle = true
	})

	return &Uploader{client: client, bucket: bucket}, nil
}

// StageRaw writes bytes to key (staging prefix e.g. raw/{user}/{id}/original.jpg).
func (u *Uploader) StageRaw(ctx context.Context, key string, data []byte, contentType string) error {
	ct := contentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(ct),
	})
	return err
}

// Delete removes an object by key.
func (u *Uploader) Delete(ctx context.Context, key string) error {
	_, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	return err
}
