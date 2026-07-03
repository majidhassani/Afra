// Package storage wraps the MinIO client used for evidence attachments.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"casemind/internal/config"
)

type Client struct {
	mc     *minio.Client
	bucket string
}

// New connects to MinIO and ensures the bucket exists. Returns (nil, nil)
// when no endpoint is configured — attachments are then disabled.
func New(ctx context.Context, cfg config.MinIO) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, nil
	}
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) Put(ctx context.Context, objectName string, r io.Reader, size int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, objectName, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *Client) PresignedGetURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, objectName, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
