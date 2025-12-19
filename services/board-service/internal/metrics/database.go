package metrics

import (
	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

// Re-export common module constants for backwards compatibility
const (
	DBOpSelect = commonmetrics.DBOpSelect
	DBOpInsert = commonmetrics.DBOpInsert
	DBOpUpdate = commonmetrics.DBOpUpdate
	DBOpDelete = commonmetrics.DBOpDelete
	DBOpCreate = commonmetrics.DBOpCreate
	DBOpDrop   = commonmetrics.DBOpDrop
)

// InferOperationFromSQL infers operation type from SQL query
// Delegates to common module
func InferOperationFromSQL(sql string) string {
	return commonmetrics.InferOperationFromSQL(sql)
}
