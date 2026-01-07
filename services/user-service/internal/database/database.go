package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"user-service/internal/domain"
)

// DefaultWorkspaceID is the UUID used for default profiles (not tied to a specific workspace)
var DefaultWorkspaceID = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// SystemUserID is the UUID used for system-owned entities
var SystemUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Config holds database configuration
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// New creates a new database connection
func New(cfg Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Workspace{},
		&domain.WorkspaceMember{},
		&domain.UserProfile{},
		&domain.WorkspaceJoinRequest{},
		&domain.Attachment{},
	)
}

// SeedDefaultData creates required default data (system user, default workspace)
// This should be called after AutoMigrate
func SeedDefaultData(db *gorm.DB) error {
	// 1. Create system user if not exists
	var systemUser domain.User
	result := db.Where("id = ?", SystemUserID).First(&systemUser)
	if result.Error == gorm.ErrRecordNotFound {
		systemUser = domain.User{
			ID:           SystemUserID,
			Email:        "system@wealist.internal",
			PasswordHash: "", // No password for system user
			IsActive:     true,
			CreatedAt:    time.Now(),
		}
		if err := db.Create(&systemUser).Error; err != nil {
			return fmt.Errorf("failed to create system user: %w", err)
		}
		fmt.Println("Created system user")
	}

	// 2. Create default workspace if not exists
	var defaultWorkspace domain.Workspace
	result = db.Where("id = ?", DefaultWorkspaceID).First(&defaultWorkspace)
	if result.Error == gorm.ErrRecordNotFound {
		description := "Default workspace for user profiles without a specific workspace"
		defaultWorkspace = domain.Workspace{
			ID:                   DefaultWorkspaceID,
			OwnerID:              SystemUserID,
			WorkspaceName:        "_default",
			WorkspaceDescription: &description,
			IsPublic:             false,
			NeedApproved:         false,
			OnlyOwnerCanInvite:   true,
			IsActive:             false, // Hidden from normal queries
			CreatedAt:            time.Now(),
		}
		if err := db.Create(&defaultWorkspace).Error; err != nil {
			return fmt.Errorf("failed to create default workspace: %w", err)
		}
		fmt.Println("Created default workspace")
	}

	return nil
}

// NewWithRetry creates a database connection with retries (blocking)
// Returns the connected DB instance or error after maxRetries
func NewWithRetry(cfg Config, retryInterval time.Duration, maxRetries int) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = New(cfg)
		if err == nil {
			fmt.Printf("Database connected successfully (attempt %d/%d)\n", i+1, maxRetries)
			return db, nil
		}
		fmt.Printf("Failed to connect to database (attempt %d/%d), retrying in %v: %v\n", i+1, maxRetries, retryInterval, err)
		time.Sleep(retryInterval)
	}

	return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
}
