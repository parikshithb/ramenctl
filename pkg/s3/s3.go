// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// Directory permission for S3 output directories
const dirPerm = 0o750

// Profile contains S3 connection and authentication information.
type Profile struct {
	Name          string
	Bucket        string
	Region        string
	Endpoint      string
	AccessKey     string
	SecretKey     string
	CACertificate []byte
}

// Client wraps an S3 client with profile information and log.
type Client struct {
	client  *s3.Client
	profile Profile
	log     *zap.SugaredLogger
}

// NewClient creates a new S3 client for the given profile.
func NewClient(
	ctx context.Context,
	profile Profile,
	log *zap.SugaredLogger,
) (*Client, error) {
	configOptions := []func(*config.LoadOptions) error{
		config.WithRegion(profile.Region),
		config.WithBaseEndpoint(profile.Endpoint),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			profile.AccessKey,
			profile.SecretKey,
			"",
		)),
	}

	// Add CA certificate if provided
	if len(profile.CACertificate) > 0 {
		log.Debugf("Using custom CA certificate for S3 profile %q", profile.Name)
		configOptions = append(
			configOptions,
			config.WithCustomCABundle(bytes.NewReader(profile.CACertificate)),
		)
	}

	config, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load s3 config for profile %q: %w",
			profile.Name, err)
	}

	client := s3.NewFromConfig(config, func(o *s3.Options) {
		// Use path-style addressing(https://endpoint/bucket/key) instead of
		// virtual-hosted style(https://bucket.endpoint/key).
		o.UsePathStyle = true
	})

	return &Client{
		client:  client,
		profile: profile,
		log:     log,
	}, nil
}

// DownloadObjects downloads all objects with the given prefix to the output directory.
func (c *Client) DownloadObjects(ctx context.Context, prefix, outputDir string) error {
	keys, err := c.listObjects(ctx, prefix)
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		c.log.Warnf("No objects found in bucket %q with prefix %q", c.profile.Bucket, prefix)
		return nil
	}

	c.log.Debugf("Downloading %d objects from bucket %q", len(keys), c.profile.Bucket)

	profileDir := filepath.Join(outputDir, "s3", c.profile.Name)
	if err := os.MkdirAll(profileDir, 0o750); err != nil {
		return err
	}

	downloaded := 0
	for _, key := range keys {
		if err := c.downloadObject(ctx, key, outputDir); err != nil {
			c.log.Warnf("Failed to download object %q from bucket %q: %v",
				key, c.profile.Bucket, err)
			continue
		}
		downloaded++
	}

	c.log.Debugf("Downloaded %d/%d objects from bucket %q",
		downloaded, len(keys), c.profile.Bucket)

	return nil
}

// listObjects returns all object keys under the given prefix.
func (c *Client) listObjects(ctx context.Context, prefix string) ([]string, error) {
	c.log.Debugf("Listing objects in bucket %q with prefix %q", c.profile.Bucket, prefix)

	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(c.profile.Bucket),
		Prefix: aws.String(prefix),
	})

	var keys []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in bucket %q with prefix %q: %w",
				c.profile.Bucket, prefix, err)
		}
		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}

	c.log.Debugf("Found %d objects in bucket %q with prefix %q",
		len(keys), c.profile.Bucket, prefix)

	return keys, nil
}

// downloadObject downloads an object from S3 store to the local filesystem.
func (c *Client) downloadObject(ctx context.Context, key string, profileDir string) error {
	objectPath := filepath.Join(profileDir, key)
	if err := os.MkdirAll(filepath.Dir(objectPath), dirPerm); err != nil {
		return err
	}

	result, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.profile.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to get object %q from bucket %q: %w", key, c.profile.Bucket, err)
	}
	defer result.Body.Close()

	file, err := os.Create(objectPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, result.Body); err != nil {
		return fmt.Errorf("failed to write object %q to file: %w", key, err)
	}

	c.log.Debugf("Downloaded object %q to %q", key, objectPath)

	return nil
}
