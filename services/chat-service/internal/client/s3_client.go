// Package client provides external service client implementations.
package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	appConfig "chat-service/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3ClientInterface defines the interface for S3 operations
type S3ClientInterface interface {
	GenerateFileKey(workspaceID, fileExt string) (string, error)
	GeneratePresignedURL(ctx context.Context, workspaceID, fileName, contentType string) (string, string, error)
	GetFileURL(key string) string
}

// S3Client wraps AWS S3 client and implements S3ClientInterface
type S3Client struct {
	client         *s3.Client
	presignClient  *s3.Client // presigned URL 생성용 별도 클라이언트
	bucket         string
	region         string
	endpoint       string // 내부 통신용 엔드포인트
	publicEndpoint string // 브라우저 접근용 공개 엔드포인트 (presigned URL용)
}

// NewS3Client creates a new S3 client for chat-service
func NewS3Client(cfg *appConfig.S3Config) (*S3Client, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("S3 region is required")
	}

	ctx := context.Background()

	// Create AWS config
	var awsCfg aws.Config
	var err error

	// If endpoint is provided (for local MinIO), use custom endpoint resolver with explicit credentials
	if cfg.Endpoint != "" {
		// MinIO requires explicit credentials
		if cfg.AccessKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("access key and secret key are required for MinIO endpoint")
		}

		// MinIO configuration with internal endpoint
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
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
		publicAwsCfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
			config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
				func(service, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
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
		if cfg.PublicEndpoint == "" {
			fmt.Fprintf(os.Stderr, "WARNING: S3_PUBLIC_ENDPOINT not set. Presigned URLs will use internal endpoint.\n")
		}
		presignClient = client
	}

	// Log the endpoints being used
	fmt.Fprintf(os.Stderr, "INFO: S3Client (chat-service) initialized - Bucket: %s, Endpoint: %s, PublicEndpoint: %s\n",
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

// GenerateFileKey generates a unique S3 file key for chat files
// Format: chat/{workspaceId}/{year}/{month}/{uuid}_{timestamp}.ext
func (c *S3Client) GenerateFileKey(workspaceID, fileExt string) (string, error) {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	fileUUID := uuid.New().String()
	timestamp := now.Unix()

	key := fmt.Sprintf("chat/%s/%s/%s/%s_%d%s",
		workspaceID, year, month, fileUUID, timestamp, fileExt)

	return key, nil
}

// GeneratePresignedURL generates a presigned URL for uploading a file to S3
// The URL expires in 5 minutes
func (c *S3Client) GeneratePresignedURL(ctx context.Context, workspaceID, fileName, contentType string) (string, string, error) {
	// Extract file extension
	fileExt := ""
	for i := len(fileName) - 1; i >= 0; i-- {
		if fileName[i] == '.' {
			fileExt = fileName[i:]
			break
		}
	}

	// Generate file key
	fileKey, err := c.GenerateFileKey(workspaceID, fileExt)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate file key: %w", err)
	}

	// Use presignClient which is configured with public endpoint
	presignClient := s3.NewPresignClient(c.presignClient)

	// Create presigned PUT request
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(fileKey),
		ContentType: aws.String(contentType),
	}

	// Generate presigned URL with 5 minute expiration
	presignedReq, err := presignClient.PresignPutObject(ctx, putObjectInput, func(opts *s3.PresignOptions) {
		opts.Expires = 5 * time.Minute
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedReq.URL, fileKey, nil
}

// GetFileURL returns the public URL for a file
func (c *S3Client) GetFileURL(key string) string {
	// CDN mode: publicEndpoint가 CloudFront 등 CDN 도메인인 경우 (s3.amazonaws.com이 아닌 경우)
	// CDN은 이미 버킷을 가리키므로 key만 추가
	if c.publicEndpoint != "" && c.endpoint == "" && !strings.Contains(c.publicEndpoint, "s3.amazonaws.com") {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), key)
	}

	// AWS S3 path-style: publicEndpoint가 s3.amazonaws.com 형태인 경우 버킷 포함
	if c.publicEndpoint != "" && c.endpoint == "" && strings.Contains(c.publicEndpoint, "s3.amazonaws.com") {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), c.bucket, key)
	}

	// MinIO 환경인 경우 (publicEndpoint가 설정된 경우 우선 사용)
	if c.publicEndpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), c.bucket, key)
	}

	// MinIO 환경 (publicEndpoint 없이 endpoint만 있는 경우)
	if c.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.endpoint, "/"), c.bucket, key)
	}

	// AWS S3 환경인 경우 (기본 - virtual-hosted style)
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
}
