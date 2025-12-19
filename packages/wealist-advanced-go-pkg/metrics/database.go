package metrics

import (
	"database/sql"
	"strings"
	"time"
)

// UpdateDBStats updates database connection pool metrics
func (m *Metrics) UpdateDBStats(statsInterface interface{}) {
	m.safeExecute("UpdateDBStats", func() {
		stats, ok := statsInterface.(sql.DBStats)
		if !ok {
			return
		}
		m.DBConnectionsOpen.Set(float64(stats.OpenConnections))
		m.DBConnectionsInUse.Set(float64(stats.InUse))
		m.DBConnectionsIdle.Set(float64(stats.Idle))
		m.DBConnectionsMax.Set(float64(stats.MaxOpenConnections))
		m.DBConnectionWaitTotal.Add(float64(stats.WaitCount))
		m.DBConnectionWaitDuration.Add(stats.WaitDuration.Seconds())
	})
}

// RecordDBQuery records database query metrics
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration, err error) {
	m.safeExecute("RecordDBQuery", func() {
		operation = NormalizeOperation(operation)
		m.DBQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())

		if err != nil {
			m.DBQueryErrors.WithLabelValues(operation, table).Inc()
		}
	})
}

// NormalizeOperation converts operation to lowercase
func NormalizeOperation(op string) string {
	return strings.ToLower(op)
}

// DBOperation constants for common database operations
const (
	DBOpSelect = "select"
	DBOpInsert = "insert"
	DBOpUpdate = "update"
	DBOpDelete = "delete"
	DBOpCreate = "create"
	DBOpDrop   = "drop"
)

// InferOperationFromSQL infers operation type from SQL query
func InferOperationFromSQL(sql string) string {
	sql = strings.TrimSpace(strings.ToLower(sql))

	switch {
	case strings.HasPrefix(sql, "select"):
		return DBOpSelect
	case strings.HasPrefix(sql, "insert"):
		return DBOpInsert
	case strings.HasPrefix(sql, "update"):
		return DBOpUpdate
	case strings.HasPrefix(sql, "delete"):
		return DBOpDelete
	case strings.HasPrefix(sql, "create"):
		return DBOpCreate
	case strings.HasPrefix(sql, "drop"):
		return DBOpDrop
	default:
		return "other"
	}
}
