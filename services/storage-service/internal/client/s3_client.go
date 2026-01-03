package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	internalConfig "storage-service/internal/config"
)

// S3Client handles S3 operations
type S3Client struct {
	client         *s3.Client
	presignClient  *s3.Client
	bucket         string
	region         string
	endpoint       string
	publicEndpoint string
}

// NewS3Client creates a new S3Client
func NewS3Client(cfg *internalConfig.S3Config) (*S3Client, error) {
	var awsCfg aws.Config
	var err error

	ctx := context.Background()

	// Check if using local MinIO (endpoint is set)
	if cfg.Endpoint != "" {
		// MinIO configuration with internal endpoint
		//nolint:staticcheck // AWS SDK v2 endpoint resolver deprecation - TODO: migrate to BaseEndpoint
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) { //nolint:staticcheck
					return aws.Endpoint{ //nolint:staticcheck
						URL:               cfg.Endpoint,
						HostnameImmutable: true,
						SigningRegion:     cfg.Region,
					}, nil
				},
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	} else {
		// AWS S3 configuration (uses IAM role or default credentials)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
	}

	// Create S3 client for internal operations
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.UsePathStyle = true // Required for MinIO
		}
	})

	// Create separate presign client with public endpoint for browser-accessible URLs
	var presignClient *s3.Client
	if cfg.PublicEndpoint != "" && cfg.Endpoint != "" {
		//nolint:staticcheck // AWS SDK v2 endpoint resolver deprecation - TODO: migrate to BaseEndpoint
		publicAwsCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) { //nolint:staticcheck
					return aws.Endpoint{ //nolint:staticcheck
						URL:               cfg.PublicEndpoint,
						HostnameImmutable: true,
						SigningRegion:     cfg.Region,
					}, nil
				},
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load public AWS config: %w", err)
		}
		presignClient = s3.NewFromConfig(publicAwsCfg, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	} else {
		presignClient = client
	}

	return &S3Client{
		client:         client,
		presignClient:  presignClient,
		bucket:         cfg.Bucket,
		region:         cfg.Region,
		endpoint:       cfg.Endpoint,
		publicEndpoint: cfg.PublicEndpoint,
	}, nil
}

// GeneratePresignedURL generates a presigned URL for uploading a file
func (c *S3Client) GeneratePresignedURL(ctx context.Context, workspaceID, fileName, contentType string) (string, string, error) {
	// Generate unique file key with workspace prefix
	fileKey := fmt.Sprintf("storage/%s/%s/%s", workspaceID, uuid.New().String(), fileName)

	// Use presignClient which is configured with public endpoint
	presignClient := s3.NewPresignClient(c.presignClient)

	// Generate presigned PUT URL
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}

	presignedReq, err := presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = 5 * time.Minute
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, fileKey, nil
}

// GenerateDownloadURL generates a presigned URL for downloading a file
func (c *S3Client) GenerateDownloadURL(ctx context.Context, fileKey, fileName string) (string, error) {
	presignClient := s3.NewPresignClient(c.presignClient)

	getObjectInput := &s3.GetObjectInput{
		Bucket:                     aws.String(c.bucket),
		Key:                        aws.String(fileKey),
		ResponseContentDisposition: aws.String(fmt.Sprintf("attachment; filename=\"%s\"", fileName)),
	}

	presignedReq, err := presignClient.PresignGetObject(ctx, getObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = 1 * time.Hour
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return presignedReq.URL, nil
}

// GetFileURL returns the public URL for a file
func (c *S3Client) GetFileURL(fileKey string) string {
	// CDN mode: publicEndpoint is set but endpoint is empty (AWS S3 + CloudFront)
	// CloudFront origin is the S3 bucket, so bucket name is not in the URL
	if c.publicEndpoint != "" && c.endpoint == "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), fileKey)
	}

	// MinIO with publicEndpoint (bucket name required in URL)
	if c.publicEndpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), c.bucket, fileKey)
	}

	// MinIO with endpoint fallback
	if c.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.endpoint, "/"), c.bucket, fileKey)
	}

	// AWS S3 direct (no CDN)
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, fileKey)
}

// DeleteFile deletes a file from S3
func (c *S3Client) DeleteFile(ctx context.Context, fileKey string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// CopyFile copies a file within S3
func (c *S3Client) CopyFile(ctx context.Context, sourceKey, destKey string) error {
	_, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(c.bucket),
		CopySource: aws.String(fmt.Sprintf("%s/%s", c.bucket, sourceKey)),
		Key:        aws.String(destKey),
	})
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	return nil
}

// FileExists checks if a file exists in S3
func (c *S3Client) FileExists(ctx context.Context, fileKey string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		// Check if error is "not found"
		return false, nil
	}
	return true, nil
}
