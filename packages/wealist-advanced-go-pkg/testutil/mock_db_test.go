package testutil

import (
	"testing"

	"github.com/google/uuid"
)

type TestModel struct {
	ID   uuid.UUID `gorm:"type:text;primaryKey"`
	Name string
}

func TestSetupTestDB(t *testing.T) {
	db, cleanup := SetupTestDB(t, nil)
	defer cleanup()

	if db == nil {
		t.Error("Expected non-nil database")
	}

	// Verify we can execute queries
	var result int
	err := db.Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestSetupTestDBWithModels(t *testing.T) {
	db, cleanup := SetupTestDBWithModels(t, &TestModel{})
	defer cleanup()

	// Verify table was created
	if !db.Migrator().HasTable(&TestModel{}) {
		t.Error("Expected TestModel table to exist")
	}

	// Test CRUD
	model := &TestModel{
		ID:   uuid.New(),
		Name: "test",
	}

	if err := db.Create(model).Error; err != nil {
		t.Errorf("Failed to create model: %v", err)
	}

	var found TestModel
	if err := db.First(&found, "id = ?", model.ID).Error; err != nil {
		t.Errorf("Failed to find model: %v", err)
	}

	if found.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", found.Name)
	}
}

func TestSetupTestDBWithLogging(t *testing.T) {
	db, cleanup := SetupTestDBWithLogging(t, &TestModel{})
	defer cleanup()

	if db == nil {
		t.Error("Expected non-nil database")
	}
}

func TestCreateTestTable(t *testing.T) {
	db, cleanup := SetupTestDB(t, nil)
	defer cleanup()

	CreateTestTable(t, db, "test_table", `
		CREATE TABLE IF NOT EXISTS test_table (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)

	// Verify table exists
	var count int
	err := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count).Error
	if err != nil {
		t.Errorf("Failed to check table: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected table to exist, count: %d", count)
	}
}

func TestDropTestTable(t *testing.T) {
	db, cleanup := SetupTestDB(t, nil)
	defer cleanup()

	// Create then drop
	db.Exec("CREATE TABLE drop_test (id TEXT)")
	DropTestTable(t, db, "drop_test")

	var count int
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='drop_test'").Scan(&count)
	if count != 0 {
		t.Error("Expected table to be dropped")
	}
}

func TestTruncateTable(t *testing.T) {
	db, cleanup := SetupTestDBWithModels(t, &TestModel{})
	defer cleanup()

	// Insert data
	db.Create(&TestModel{ID: uuid.New(), Name: "test1"})
	db.Create(&TestModel{ID: uuid.New(), Name: "test2"})

	// Truncate
	TruncateTable(t, db, "test_models")

	// Verify empty
	var count int64
	db.Model(&TestModel{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected 0 rows after truncate, got %d", count)
	}
}
