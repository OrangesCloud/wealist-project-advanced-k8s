// Package testutil provides common testing utilities for Go microservices.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig holds configuration for test database setup
type DBConfig struct {
	// UseTempFile uses a temporary file instead of :memory: for debugging
	UseTempFile bool
	// LogLevel sets the GORM log level (default: Silent)
	LogLevel logger.LogLevel
	// Models to auto-migrate
	Models []interface{}
}

// DefaultDBConfig returns default test database configuration
func DefaultDBConfig() *DBConfig {
	return &DBConfig{
		UseTempFile: false,
		LogLevel:    logger.Silent,
		Models:      nil,
	}
}

// SetupTestDB creates an in-memory SQLite database for testing.
// It automatically registers UUID generation callback and returns a cleanup function.
func SetupTestDB(t *testing.T, config *DBConfig) (*gorm.DB, func()) {
	t.Helper()

	if config == nil {
		config = DefaultDBConfig()
	}

	var dsn string
	var tempFile string

	if config.UseTempFile {
		tempDir := os.TempDir()
		tempFile = filepath.Join(tempDir, fmt.Sprintf("test_%s.db", uuid.New().String()))
		dsn = tempFile
	} else {
		dsn = ":memory:"
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(config.LogLevel),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Register UUID generation callback for SQLite
	RegisterUUIDCallback(db)

	// Auto-migrate models if provided
	if len(config.Models) > 0 {
		if err := db.AutoMigrate(config.Models...); err != nil {
			t.Fatalf("Failed to auto-migrate models: %v", err)
		}
	}

	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
		if tempFile != "" {
			_ = os.Remove(tempFile)
		}
	}

	return db, cleanup
}

// RegisterUUIDCallback registers a GORM callback that generates UUIDs for models
// before creating them. This is necessary for SQLite which doesn't support
// gen_random_uuid() like PostgreSQL.
func RegisterUUIDCallback(db *gorm.DB) {
	_ = db.Callback().Create().Before("gorm:create").Register("uuid_generate", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil {
			for _, field := range tx.Statement.Schema.Fields {
				if field.Name == "ID" && field.FieldType.String() == "uuid.UUID" {
					fieldValue := tx.Statement.ReflectValue
					if fieldValue.IsValid() {
						idField := fieldValue.FieldByName("ID")
						if idField.IsValid() && idField.CanSet() {
							// Check if ID is zero value
							currentID := idField.Interface()
							if zeroUUID, ok := currentID.(uuid.UUID); ok && zeroUUID == uuid.Nil {
								idField.Set(fieldValue.FieldByName("ID").Addr().Elem())
								newUUID := uuid.New()
								idField.SetBytes(newUUID[:])
							}
						}
					}
				}
			}
		}
	})
}

// SetupTestDBWithModels is a convenience function that creates a test database
// and auto-migrates the provided models.
func SetupTestDBWithModels(t *testing.T, models ...interface{}) (*gorm.DB, func()) {
	t.Helper()
	config := DefaultDBConfig()
	config.Models = models
	return SetupTestDB(t, config)
}

// SetupTestDBWithLogging creates a test database with logging enabled.
// Useful for debugging test failures.
func SetupTestDBWithLogging(t *testing.T, models ...interface{}) (*gorm.DB, func()) {
	t.Helper()
	config := &DBConfig{
		UseTempFile: false,
		LogLevel:    logger.Info,
		Models:      models,
	}
	return SetupTestDB(t, config)
}

// CreateTestTable creates a table using raw SQL.
// This is useful when AutoMigrate doesn't work due to SQLite limitations.
func CreateTestTable(t *testing.T, db *gorm.DB, tableName, createSQL string) {
	t.Helper()
	if err := db.Exec(createSQL).Error; err != nil {
		t.Fatalf("Failed to create table %s: %v", tableName, err)
	}
}

// DropTestTable drops a table if it exists.
func DropTestTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Error; err != nil {
		t.Fatalf("Failed to drop table %s: %v", tableName, err)
	}
}

// TruncateTable truncates a table (deletes all rows).
func TruncateTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	if err := db.Exec(fmt.Sprintf("DELETE FROM %s", tableName)).Error; err != nil {
		t.Fatalf("Failed to truncate table %s: %v", tableName, err)
	}
}
