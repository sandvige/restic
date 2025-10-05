// Package resticlib provides a Go library API for restic backup functionality.
// This package extracts the core restic functionality to be used as a library,
// separate from the CLI interface.
package resticlib

import (
	"context"
	"fmt"
	"io"
)

// BackendKind represents the type of storage backend
type BackendKind string

const (
	BackendLocal BackendKind = "local"
	BackendS3    BackendKind = "s3" 
	BackendAzure BackendKind = "azure"
	BackendGCS   BackendKind = "gcs"
	BackendB2    BackendKind = "b2"
	BackendSFTP  BackendKind = "sftp"
	BackendSwift BackendKind = "swift"
	BackendRest  BackendKind = "rest"
)

// Credentials holds authentication information for backends
type Credentials struct {
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
	Token     string `json:"token,omitempty"`
}

// Logger interface for pluggable logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// ProgressReporter interface for progress callbacks  
type ProgressReporter interface {
	SetTotal(total uint64)
	Add(delta uint64)
	Error(item string, err error) error
	Finish()
}

// Config holds configuration for repository operations
type Config struct {
	// RepoURL is the repository location (e.g., "s3:s3.amazonaws.com/bucket/path")
	RepoURL string

	// Backend specifies the storage backend type
	Backend BackendKind

	// Credentials for backend authentication (optional)
	Credentials *Credentials

	// Password for repository encryption (never logged)
	Password []byte

	// CACertsPEM for custom CA certificates (optional)
	CACertsPEM []byte

	// Parallelism controls number of workers for upload/download
	Parallelism int

	// TempDir for temporary files (optional, defaults to system temp)
	TempDir string

	// Logger for log output (optional)
	Logger Logger
}

// SnapshotID represents a unique snapshot identifier
type SnapshotID string

// String returns the string representation of the snapshot ID
func (s SnapshotID) String() string {
	return string(s)
}

// Snapshot contains metadata about a backup snapshot
type Snapshot struct {
	ID       SnapshotID `json:"id"`
	Time     string     `json:"time"`
	Tree     string     `json:"tree"`
	Paths    []string   `json:"paths"`
	Hostname string     `json:"hostname"`
	Username string     `json:"username"`
	Tags     []string   `json:"tags,omitempty"`
	Parent   *string    `json:"parent,omitempty"`
	Summary  *struct {
		FilesNew          uint64 `json:"files_new"`
		FilesChanged      uint64 `json:"files_changed"`
		FilesUnmodified   uint64 `json:"files_unmodified"`
		DirsNew           uint64 `json:"dirs_new"`
		DirsChanged       uint64 `json:"dirs_changed"`
		DirsUnmodified    uint64 `json:"dirs_unmodified"`
		DataBlobs         uint64 `json:"data_blobs"`
		TreeBlobs         uint64 `json:"tree_blobs"`
		DataAdded         uint64 `json:"data_added"`
		TotalFilesProcessed uint64 `json:"total_files_processed"`
		TotalBytesProcessed uint64 `json:"total_bytes_processed"`
		TotalDuration     float64 `json:"total_duration"`
		SnapshotID        string  `json:"snapshot_id"`
	} `json:"summary,omitempty"`
}

// BackupOptions configures backup operations
type BackupOptions struct {
	Paths        []string         `json:"paths"`
	Tags         []string         `json:"tags,omitempty"`
	Excludes     []string         `json:"excludes,omitempty"`
	Includes     []string         `json:"includes,omitempty"`
	ParentID     *SnapshotID      `json:"parent_id,omitempty"`
	DryRun       bool             `json:"dry_run,omitempty"`
	Progress     ProgressReporter `json:"-"`
}

// RestoreOptions configures restore operations  
type RestoreOptions struct {
	TargetDir string           `json:"target_dir"`
	Includes  []string         `json:"includes,omitempty"`
	Excludes  []string         `json:"excludes,omitempty"`
	Overwrite bool             `json:"overwrite,omitempty"`
	Delete    bool             `json:"delete,omitempty"`
	DryRun    bool             `json:"dry_run,omitempty"`
	Progress  ProgressReporter `json:"-"`
}

// SnapshotFilter for filtering snapshots
type SnapshotFilter struct {
	Hosts []string    `json:"hosts,omitempty"`
	Paths []string    `json:"paths,omitempty"`
	Tags  []string    `json:"tags,omitempty"`
	Since *string     `json:"since,omitempty"`
	Until *string     `json:"until,omitempty"`
	Limit int         `json:"limit,omitempty"`
}

// ForgetPolicy defines retention policy for snapshots
type ForgetPolicy struct {
	KeepLast    int      `json:"keep_last,omitempty"`
	KeepHourly  int      `json:"keep_hourly,omitempty"`
	KeepDaily   int      `json:"keep_daily,omitempty"`
	KeepWeekly  int      `json:"keep_weekly,omitempty"`
	KeepMonthly int      `json:"keep_monthly,omitempty"`
	KeepYearly  int      `json:"keep_yearly,omitempty"`
	KeepWithin  *string  `json:"keep_within,omitempty"`
	KeepTags    []string `json:"keep_tags,omitempty"`
}

// Empty returns true if the policy has no rules set
func (p ForgetPolicy) Empty() bool {
	return p.KeepLast == 0 && p.KeepHourly == 0 && p.KeepDaily == 0 &&
		p.KeepWeekly == 0 && p.KeepMonthly == 0 && p.KeepYearly == 0 &&
		p.KeepWithin == nil && len(p.KeepTags) == 0
}

// PruneOptions configures prune operations
type PruneOptions struct {
	DryRun        bool             `json:"dry_run,omitempty"`
	MaxUnused     string           `json:"max_unused,omitempty"`
	MaxRepackSize string           `json:"max_repack_size,omitempty"`
	Progress      ProgressReporter `json:"-"`
}

// PruneReport contains results of prune operation
type PruneReport struct {
	PacksDeleted   int    `json:"packs_deleted"`
	PacksKept      int    `json:"packs_kept"`
	PacksRepacked  int    `json:"packs_repacked"`
	BytesDeleted   uint64 `json:"bytes_deleted"`
	BytesRepacked  uint64 `json:"bytes_repacked"`
}

// CheckDepth controls how thorough the integrity check is
type CheckDepth string

const (
	CheckDepthDefault  CheckDepth = "default"
	CheckDepthFull     CheckDepth = "full"
	CheckDepthReadData CheckDepth = "read_data"
)

// CheckReport contains results of integrity check
type CheckReport struct {
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Success  bool     `json:"success"`
}

// Repository interface provides access to a restic repository
type Repository interface {
	// Backup creates a new backup snapshot
	Backup(ctx context.Context, opts BackupOptions) (SnapshotID, error)

	// Restore restores files from a snapshot 
	Restore(ctx context.Context, snapshotID SnapshotID, opts RestoreOptions) error

	// Snapshots lists snapshots matching the filter
	Snapshots(ctx context.Context, filter SnapshotFilter) ([]Snapshot, error)

	// Forget removes snapshots according to policy
	Forget(ctx context.Context, policy ForgetPolicy) ([]SnapshotID, error)

	// Prune removes unused data from repository
	Prune(ctx context.Context, opts PruneOptions) (PruneReport, error)

	// Check verifies repository integrity
	Check(ctx context.Context, depth CheckDepth) (CheckReport, error)

	// Unlock removes stale locks from repository
	Unlock(ctx context.Context) error

	// Close closes the repository connection
	Close() error
}



// DefaultLogger provides a simple logger that writes to the provided writer
type DefaultLogger struct {
	Writer io.Writer
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...interface{}) {
	// Debug messages are typically not shown in default logger
}

// Info logs an info message  
func (l *DefaultLogger) Info(msg string, args ...interface{}) {
	if l.Writer != nil {
		_, _ = l.Writer.Write([]byte(fmt.Sprintf("[INFO] "+msg+"\n", args...)))
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...interface{}) {
	if l.Writer != nil {
		_, _ = l.Writer.Write([]byte(fmt.Sprintf("[WARN] "+msg+"\n", args...)))
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...interface{}) {
	if l.Writer != nil {
		_, _ = l.Writer.Write([]byte(fmt.Sprintf("[ERROR] "+msg+"\n", args...)))
	}
}