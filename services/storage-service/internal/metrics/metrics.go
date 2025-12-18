// Package metrics provides Prometheus metrics for storage-service.
//
// This package extends the common metrics package with business-specific metrics
// for tracking storage operations, files, folders, and projects.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	commonmetrics "github.com/OrangesCloud/wealist-advanced-go-pkg/metrics"
)

const namespace = "storage_service"

// Metrics holds all application metrics for storage-service.
type Metrics struct {
	// Embedded common metrics for HTTP requests, database operations, etc.
	*commonmetrics.Metrics

	// FilesTotal tracks the current number of files.
	FilesTotal prometheus.Gauge
	// FoldersTotal tracks the current number of folders.
	FoldersTotal prometheus.Gauge
	// ProjectsTotal tracks the current number of storage projects.
	ProjectsTotal prometheus.Gauge
	// SharesTotal tracks the current number of active shares.
	SharesTotal prometheus.Gauge

	// FileUploadTotal counts file upload operations.
	FileUploadTotal prometheus.Counter
	// FileDownloadTotal counts file download operations.
	FileDownloadTotal prometheus.Counter
	// FileDeleteTotal counts file delete operations.
	FileDeleteTotal prometheus.Counter

	// StorageBytesTotal tracks total storage used in bytes.
	StorageBytesTotal prometheus.Gauge
}

// New creates and registers all metrics with the default Prometheus registerer.
func New() *Metrics {
	return NewWithRegistry(prometheus.DefaultRegisterer)
}

// NewWithRegistry creates metrics with a custom registry.
func NewWithRegistry(registerer prometheus.Registerer) *Metrics {
	cfg := &commonmetrics.Config{
		Namespace: namespace,
		Registry:  registerer,
	}

	factory := promauto.With(registerer)

	return &Metrics{
		Metrics: commonmetrics.New(cfg),

		FilesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "files_total",
				Help:      "Total number of files",
			},
		),
		FoldersTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "folders_total",
				Help:      "Total number of folders",
			},
		),
		ProjectsTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "projects_total",
				Help:      "Total number of storage projects",
			},
		),
		SharesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "shares_total",
				Help:      "Total number of active shares",
			},
		),
		FileUploadTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_upload_total",
				Help:      "Total number of file uploads",
			},
		),
		FileDownloadTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_download_total",
				Help:      "Total number of file downloads",
			},
		),
		FileDeleteTotal: factory.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "file_delete_total",
				Help:      "Total number of file deletions",
			},
		),
		StorageBytesTotal: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "storage_bytes_total",
				Help:      "Total storage used in bytes",
			},
		),
	}
}

// NewForTest creates metrics with an isolated registry for testing.
func NewForTest() *Metrics {
	return NewWithRegistry(prometheus.NewRegistry())
}

// RecordFileUpload increments file upload counter.
func (m *Metrics) RecordFileUpload() {
	m.FileUploadTotal.Inc()
}

// RecordFileDownload increments file download counter.
func (m *Metrics) RecordFileDownload() {
	m.FileDownloadTotal.Inc()
}

// RecordFileDelete increments file delete counter.
func (m *Metrics) RecordFileDelete() {
	m.FileDeleteTotal.Inc()
}

// SetFilesTotal sets the total number of files.
func (m *Metrics) SetFilesTotal(count int64) {
	m.FilesTotal.Set(float64(count))
}

// SetFoldersTotal sets the total number of folders.
func (m *Metrics) SetFoldersTotal(count int64) {
	m.FoldersTotal.Set(float64(count))
}

// SetProjectsTotal sets the total number of projects.
func (m *Metrics) SetProjectsTotal(count int64) {
	m.ProjectsTotal.Set(float64(count))
}

// SetSharesTotal sets the total number of shares.
func (m *Metrics) SetSharesTotal(count int64) {
	m.SharesTotal.Set(float64(count))
}

// SetStorageBytesTotal sets the total storage used.
func (m *Metrics) SetStorageBytesTotal(bytes int64) {
	m.StorageBytesTotal.Set(float64(bytes))
}
