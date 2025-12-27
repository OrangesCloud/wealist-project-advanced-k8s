package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"

	internalConfig "user-service/internal/config"
)

// S3Client handles S3 operations
type S3Client struct {
	client         *s3.Client
	presignClient  *s3.Client  // Separate client for presigned URL generation with public endpoint
	bucket         string
	region         string
	endpoint       string // MinIO 내부 통신용 엔드포인트
	publicEndpoint string // 브라우저 접근용 공개 엔드포인트
}

// NewS3Client creates a new S3Client
func NewS3Client(cfg *internalConfig.S3Config) (*S3Client, error) {
	var awsCfg aws.Config
	var err error

	ctx := context.Background()

	// Check if using local MinIO (endpoint is set)
	if cfg.Endpoint != "" {
		// MinIO configuration with internal endpoint
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			//nolint:staticcheck // AWS SDK v2 endpoint resolver deprecation - TODO: migrate to BaseEndpoint
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
	// This ensures presigned URLs are signed with the correct host that browsers will use
	var presignClient *s3.Client
	if cfg.PublicEndpoint != "" && cfg.Endpoint != "" {
		publicAwsCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			//nolint:staticcheck // AWS SDK v2 endpoint resolver deprecation - TODO: migrate to BaseEndpoint
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
			// Log warning but don't fail - fallback to regular client
			fmt.Fprintf(os.Stderr, "WARNING: Failed to load public AWS config: %v. Using internal endpoint for presigned URLs.\n", err)
			presignClient = client
		} else {
			presignClient = s3.NewFromConfig(publicAwsCfg, func(o *s3.Options) {
				o.UsePathStyle = true
			})
		}
	} else {
		// Log warning about missing configuration
		if cfg.PublicEndpoint == "" {
			fmt.Fprintf(os.Stderr, "WARNING: S3_PUBLIC_ENDPOINT not set. Presigned URLs will use internal endpoint.\n")
		}
		presignClient = client
	}

	// Log the endpoints being used
	fmt.Fprintf(os.Stderr, "INFO: S3Client initialized - Bucket: %s, Endpoint: %s, PublicEndpoint: %s\n",
		cfg.Bucket, cfg.Endpoint, cfg.PublicEndpoint)

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
func (c *S3Client) GeneratePresignedURL(ctx context.Context, fileName, contentType string) (string, string, error) {
	// Generate unique file key
	fileKey := fmt.Sprintf("profiles/%s/%s", uuid.New().String(), fileName)

	// Use presignClient which is configured with public endpoint
	// This ensures the signature is computed against the URL the browser will actually use
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

// GetFileURL returns the public URL for a file
func (c *S3Client) GetFileURL(fileKey string) string {
	// MinIO 환경인 경우 - publicEndpoint 사용 (브라우저 접근용)
	if c.publicEndpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), c.bucket, fileKey)
	}

	// MinIO 환경이지만 publicEndpoint가 없는 경우 - endpoint fallback
	if c.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.endpoint, "/"), c.bucket, fileKey)
	}

	// AWS S3 환경인 경우 (기본)
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
