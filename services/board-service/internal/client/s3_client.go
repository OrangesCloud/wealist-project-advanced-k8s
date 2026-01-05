// Package client provides external service client implementations.
package client

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	appConfig "project-board-api/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// S3ClientInterface defines the interface for S3 operations
type S3ClientInterface interface {
	GenerateFileKey(entityType, workspaceID, fileExt string) (string, error)
	GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error)
	UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error)
	DeleteFile(ctx context.Context, key string) error
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

// NewS3Client creates a new S3 client
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

// GenerateFileKey generates a unique S3 file key
// Format: board/{entityType}/{workspaceId}/{year}/{month}/{uuid}_{timestamp}.ext
// entityType: "boards", "comments", "projects"
func (c *S3Client) GenerateFileKey(entityType, workspaceID, fileExt string) (string, error) {
	// Validate entityType
	validTypes := map[string]bool{
		"boards":   true,
		"comments": true,
		"projects": true,
	}
	if !validTypes[entityType] {
		return "", fmt.Errorf("invalid entity type: %s (must be 'boards', 'comments', or 'projects')", entityType)
	}

	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	fileUUID := uuid.New().String()
	timestamp := now.Unix()

	key := fmt.Sprintf("board/%s/%s/%s/%s/%s_%d%s",
		entityType, workspaceID, year, month, fileUUID, timestamp, fileExt)

	return key, nil
}

// GeneratePresignedURL generates a presigned URL for uploading a file to S3
// The URL expires in 5 minutes
func (c *S3Client) GeneratePresignedURL(ctx context.Context, entityType, workspaceID, fileName, contentType string) (string, string, error) {
	// Extract file extension
	fileExt := ""
	for i := len(fileName) - 1; i >= 0; i-- {
		if fileName[i] == '.' {
			fileExt = fileName[i:]
			break
		}
	}

	// Generate file key
	fileKey, err := c.GenerateFileKey(entityType, workspaceID, fileExt)
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

// UploadFile uploads a file to S3
func (c *S3Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) (string, error) {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Generate file URL
	fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
	return fileURL, nil
}

// DeleteFile deletes a file from S3
func (c *S3Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

// GetFileURL returns the public URL for a file
// S3 Key를 기반으로 다운로드 가능한 URL을 생성합니다.
func (c *S3Client) GetFileURL(key string) string {
	// CDN mode: publicEndpoint is set but endpoint is empty (AWS S3 + CloudFront)
	// CloudFront origin is the S3 bucket, so bucket name is not in the URL
	if c.publicEndpoint != "" && c.endpoint == "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), key)
	}

	// MinIO 환경인 경우 (publicEndpoint가 설정된 경우 우선 사용)
	if c.publicEndpoint != "" {
		// 예: https://local.wealist.co.kr/minio/bucket/key
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.publicEndpoint, "/"), c.bucket, key)
	}

	// MinIO 환경 (publicEndpoint 없이 endpoint만 있는 경우)
	if c.endpoint != "" {
		// 예: http://localhost:9000/bucket/key
		return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.endpoint, "/"), c.bucket, key)
	}

	// AWS S3 환경인 경우 (기본)
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, key)
}
