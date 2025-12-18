package metrics

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestRecordDBQuery(t *testing.T) {
	m := NewForTest("test_service")

	// Should not panic
	m.RecordDBQuery("SELECT", "users", 10*time.Millisecond, nil)
	m.RecordDBQuery("INSERT", "users", 20*time.Millisecond, nil)
	m.RecordDBQuery("UPDATE", "users", 15*time.Millisecond, errors.New("constraint violation"))
	m.RecordDBQuery("DELETE", "users", 5*time.Millisecond, nil)
}

func TestUpdateDBStats(t *testing.T) {
	m := NewForTest("test_service")

	stats := sql.DBStats{
		OpenConnections: 10,
		InUse:           5,
		Idle:            5,
		MaxOpenConnections: 20,
		WaitCount:       100,
		WaitDuration:    time.Second * 5,
	}

	// Should not panic
	m.UpdateDBStats(stats)
}

func TestUpdateDBStatsWithInvalidType(t *testing.T) {
	m := NewForTest("test_service")

	// Should not panic with invalid type
	m.UpdateDBStats("not a DBStats")
	m.UpdateDBStats(nil)
	m.UpdateDBStats(123)
}

func TestNormalizeOperation(t *testing.T) {
	tests := []struct {
		op       string
		expected string
	}{
		{"SELECT", "select"},
		{"Insert", "insert"},
		{"UPDATE", "update"},
		{"delete", "delete"},
		{"CREATE", "create"},
		{"Drop", "drop"},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			result := NormalizeOperation(tt.op)
			if result != tt.expected {
				t.Errorf("NormalizeOperation(%s) = %s, want %s", tt.op, result, tt.expected)
			}
		})
	}
}

func TestInferOperationFromSQL(t *testing.T) {
	tests := []struct {
		sql      string
		expected string
	}{
		{"SELECT * FROM users", DBOpSelect},
		{"  select id from users", DBOpSelect},
		{"INSERT INTO users (name) VALUES ('test')", DBOpInsert},
		{"insert into users values (1)", DBOpInsert},
		{"UPDATE users SET name = 'test'", DBOpUpdate},
		{"  update users set active = true", DBOpUpdate},
		{"DELETE FROM users WHERE id = 1", DBOpDelete},
		{"delete from users", DBOpDelete},
		{"CREATE TABLE users (id INT)", DBOpCreate},
		{"DROP TABLE users", DBOpDrop},
		{"TRUNCATE users", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			result := InferOperationFromSQL(tt.sql)
			if result != tt.expected {
				t.Errorf("InferOperationFromSQL(%s) = %s, want %s", tt.sql, result, tt.expected)
			}
		})
	}
}

func TestDBOperationConstants(t *testing.T) {
	// Verify constants are defined correctly
	if DBOpSelect != "select" {
		t.Errorf("DBOpSelect = %s, want select", DBOpSelect)
	}
	if DBOpInsert != "insert" {
		t.Errorf("DBOpInsert = %s, want insert", DBOpInsert)
	}
	if DBOpUpdate != "update" {
		t.Errorf("DBOpUpdate = %s, want update", DBOpUpdate)
	}
	if DBOpDelete != "delete" {
		t.Errorf("DBOpDelete = %s, want delete", DBOpDelete)
	}
	if DBOpCreate != "create" {
		t.Errorf("DBOpCreate = %s, want create", DBOpCreate)
	}
	if DBOpDrop != "drop" {
		t.Errorf("DBOpDrop = %s, want drop", DBOpDrop)
	}
}
