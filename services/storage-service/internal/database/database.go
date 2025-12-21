package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"storage-service/internal/domain"
)

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
		&domain.Project{},
		&domain.ProjectMember{},
		&domain.Folder{},
		&domain.File{},
		&domain.FileShare{},
		&domain.FolderShare{},
	)
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
