package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps an S3-compatible client for Cloudflare R2.
type Client struct {
	s3        *s3.Client
	bucket    string
	publicURL string
}

// NewClient creates a new R2-compatible S3 client.
func NewClient(accountID, accessKeyID, secretAccessKey, bucket, publicURL string) (*Client, error) {
	if accountID == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("R2 credentials are required")
	}

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID),
			HostnameImmutable: true,
			SigningRegion:     "auto",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &Client{
		s3:        client,
		bucket:    bucket,
		publicURL: publicURL,
	}, nil
}

// Upload uploads a file to R2 and returns the public URL.
func (c *Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) (string, error) {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.publicURL, key)
	return url, nil
}

// Delete removes a file from R2.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}
	return nil
}

// GenerateKey creates a unique file key for storage.
func GenerateKey(context, filename string) string {
	ext := path.Ext(filename)
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s/%d%s", context, timestamp, ext)
}
