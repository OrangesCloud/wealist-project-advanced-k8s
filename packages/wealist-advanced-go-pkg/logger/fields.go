// Package logger provides structured logging utilities.
// This file contains OpenTelemetry Semantic Conventions field constants and helper functions.
package logger

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// OpenTelemetry Semantic Conventions field names.
// See: https://opentelemetry.io/docs/specs/semconv/
const (
	// Trace context fields
	FieldTraceID      = "trace_id"
	FieldSpanID       = "span_id"
	FieldTraceSampled = "trace_sampled"

	// Service fields
	FieldServiceName    = "service.name"
	FieldServiceVersion = "service.version"
	FieldEnvironment    = "deployment.environment"

	// HTTP fields
	FieldHTTPMethod     = "http.method"
	FieldHTTPRoute      = "http.route"
	FieldHTTPStatusCode = "http.status_code"
	FieldHTTPRequestID  = "http.request_id"
	FieldHTTPClientIP   = "http.client_ip"
	FieldHTTPUserAgent  = "http.user_agent"
	FieldHTTPDuration   = "http.duration"
	FieldHTTPURL        = "http.url"
	FieldHTTPScheme     = "http.scheme"
	FieldHTTPTarget     = "http.target"

	// Database fields
	FieldDBSystem    = "db.system"
	FieldDBName      = "db.name"
	FieldDBOperation = "db.operation"
	FieldDBStatement = "db.statement"
	FieldDBTable     = "db.sql.table"
	FieldDBDuration  = "db.duration"

	// User fields
	FieldUserID    = "enduser.id"
	FieldUserRole  = "enduser.role"
	FieldUserScope = "enduser.scope"

	// Code fields
	FieldCodeFunction = "code.function"
	FieldCodeFilepath = "code.filepath"
	FieldCodeLineno   = "code.lineno"
	FieldCodeNamespace = "code.namespace"

	// RPC/gRPC fields
	FieldRPCSystem  = "rpc.system"
	FieldRPCService = "rpc.service"
	FieldRPCMethod  = "rpc.method"

	// Messaging fields
	FieldMessagingSystem      = "messaging.system"
	FieldMessagingDestination = "messaging.destination.name"
	FieldMessagingOperation   = "messaging.operation"

	// Peer/Network fields
	FieldPeerService = "peer.service"
	FieldNetPeerName = "net.peer.name"
	FieldNetPeerPort = "net.peer.port"

	// Error fields
	FieldErrorType    = "error.type"
	FieldErrorMessage = "error.message"

	// Custom wealist fields
	FieldWorkspaceID = "wealist.workspace_id"
	FieldBoardID     = "wealist.board_id"
	FieldChatID      = "wealist.chat_id"
	FieldFileID      = "wealist.file_id"
	FieldRoomID      = "wealist.room_id"
)

// Database system values
const (
	DBSystemPostgreSQL = "postgresql"
	DBSystemRedis      = "redis"
	DBSystemMySQL      = "mysql"
)

// Database operation values
const (
	DBOperationSelect = "SELECT"
	DBOperationInsert = "INSERT"
	DBOperationUpdate = "UPDATE"
	DBOperationDelete = "DELETE"
)

// HTTPRequestFields returns zap fields for an HTTP request.
// This follows OpenTelemetry HTTP semantic conventions.
func HTTPRequestFields(r *http.Request) []zap.Field {
	fields := []zap.Field{
		zap.String(FieldHTTPMethod, r.Method),
		zap.String(FieldHTTPTarget, r.URL.Path),
		zap.String(FieldHTTPScheme, r.URL.Scheme),
		zap.String(FieldHTTPUserAgent, r.UserAgent()),
	}

	if r.URL.RawQuery != "" {
		fields = append(fields, zap.String("http.query", r.URL.RawQuery))
	}

	return fields
}

// HTTPResponseFields returns zap fields for an HTTP response.
func HTTPResponseFields(statusCode int, duration time.Duration) []zap.Field {
	return []zap.Field{
		zap.Int(FieldHTTPStatusCode, statusCode),
		zap.Duration(FieldHTTPDuration, duration),
	}
}

// DBFields returns zap fields for a database operation.
func DBFields(system, operation, table string) []zap.Field {
	return []zap.Field{
		zap.String(FieldDBSystem, system),
		zap.String(FieldDBOperation, operation),
		zap.String(FieldDBTable, table),
	}
}

// DBFieldsWithDuration returns zap fields for a database operation with duration.
func DBFieldsWithDuration(system, operation, table string, duration time.Duration) []zap.Field {
	return []zap.Field{
		zap.String(FieldDBSystem, system),
		zap.String(FieldDBOperation, operation),
		zap.String(FieldDBTable, table),
		zap.Duration(FieldDBDuration, duration),
	}
}

// TraceFields extracts trace context from context and returns zap fields.
func TraceFields(ctx context.Context) []zap.Field {
	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return nil
	}

	return []zap.Field{
		zap.String(FieldTraceID, spanCtx.TraceID().String()),
		zap.String(FieldSpanID, spanCtx.SpanID().String()),
		zap.Bool(FieldTraceSampled, spanCtx.IsSampled()),
	}
}

// UserFields returns zap fields for user context.
func UserFields(userID string) []zap.Field {
	return []zap.Field{
		zap.String(FieldUserID, userID),
	}
}

// ServiceFields returns zap fields for service identification.
func ServiceFields(name, version, env string) []zap.Field {
	return []zap.Field{
		zap.String(FieldServiceName, name),
		zap.String(FieldServiceVersion, version),
		zap.String(FieldEnvironment, env),
	}
}

// PeerServiceFields returns zap fields for outbound service calls.
func PeerServiceFields(serviceName, url string) []zap.Field {
	return []zap.Field{
		zap.String(FieldPeerService, serviceName),
		zap.String(FieldHTTPURL, url),
	}
}

// ErrorFields returns zap fields for error information.
func ErrorFields(err error) []zap.Field {
	if err == nil {
		return nil
	}
	return []zap.Field{
		zap.String(FieldErrorType, "error"),
		zap.Error(err),
	}
}

// WealistBoardFields returns wealist-specific fields for board operations.
func WealistBoardFields(workspaceID, boardID string) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if workspaceID != "" {
		fields = append(fields, zap.String(FieldWorkspaceID, workspaceID))
	}
	if boardID != "" {
		fields = append(fields, zap.String(FieldBoardID, boardID))
	}
	return fields
}

// WealistChatFields returns wealist-specific fields for chat operations.
func WealistChatFields(workspaceID, chatID string) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if workspaceID != "" {
		fields = append(fields, zap.String(FieldWorkspaceID, workspaceID))
	}
	if chatID != "" {
		fields = append(fields, zap.String(FieldChatID, chatID))
	}
	return fields
}

// WealistStorageFields returns wealist-specific fields for storage operations.
func WealistStorageFields(workspaceID, fileID string) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if workspaceID != "" {
		fields = append(fields, zap.String(FieldWorkspaceID, workspaceID))
	}
	if fileID != "" {
		fields = append(fields, zap.String(FieldFileID, fileID))
	}
	return fields
}

// WealistVideoFields returns wealist-specific fields for video operations.
func WealistVideoFields(workspaceID, roomID string) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if workspaceID != "" {
		fields = append(fields, zap.String(FieldWorkspaceID, workspaceID))
	}
	if roomID != "" {
		fields = append(fields, zap.String(FieldRoomID, roomID))
	}
	return fields
}
