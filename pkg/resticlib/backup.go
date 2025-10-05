package resticlib

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/restic/restic/internal/archiver"
	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/fs"
	"github.com/restic/restic/internal/restic"
)

// archiverWrapper helps with archiver functionality
type archiverWrapper struct {
	arch     *archiver.Archiver
	reporter ProgressReporter
}

// Backup creates a new backup snapshot
func (r *repositoryImpl) Backup(ctx context.Context, opts BackupOptions) (SnapshotID, error) {
	if len(opts.Paths) == 0 {
		return "", errors.New("no paths specified for backup")
	}

	r.logf("info", "Starting backup of paths: %v", opts.Paths)

	// Load index
	err := r.repo.LoadIndex(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to load index: %w", err)
	}

	// Set up filesystem
	targetFS := fs.Local{}

	// Create archiver
	arch := archiver.New(r.repo, targetFS, archiver.Options{})

	// Set up select functions for filtering
	arch.SelectByName = func(item string) bool {
		// Apply includes first (if any)
		if len(opts.Includes) > 0 {
			matched := false
			for _, pattern := range opts.Includes {
				if matched, _ = filepath.Match(pattern, filepath.Base(item)); matched {
					break
				}
			}
			if !matched {
				return false
			}
		}

		// Apply excludes
		for _, pattern := range opts.Excludes {
			if matched, _ := filepath.Match(pattern, filepath.Base(item)); matched {
				return false
			}
		}
		return true
	}

	// Set up error handling
	arch.Error = func(file string, err error) error {
		if opts.Progress != nil {
			return opts.Progress.Error(file, err)
		}
		return err
	}

	// Set up progress reporting
	if opts.Progress != nil {
		arch.CompleteItem = func(item string, previous, current *data.Node, s archiver.ItemStats, d time.Duration) {
			opts.Progress.Add(s.DataSize + s.TreeSize)
		}
	}

	// Find parent snapshot if specified
	var parentSnapshot *data.Snapshot
	if opts.ParentID != nil {
		id, err := restic.ParseID(string(*opts.ParentID))
		if err != nil {
			return "", fmt.Errorf("invalid parent ID: %w", err)
		}
		parentSnapshot, err = data.LoadSnapshot(ctx, r.repo, id)
		if err != nil {
			return "", fmt.Errorf("failed to load parent snapshot: %w", err)
		}
	}

	// Create snapshot metadata
	hostname := "unknown"
	if h, err := os.Hostname(); err == nil {
		hostname = h
	}

	username := "unknown"
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
	_ = username // Mark as used for now

	// Resolve and clean paths
	var resolvedPaths []string
	for _, path := range opts.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path %q: %w", path, err)
		}
		resolvedPaths = append(resolvedPaths, absPath)
	}

	// Create snapshot options
	snapshotOpts := archiver.SnapshotOptions{
		Tags:           opts.Tags,
		Hostname:       hostname,
		Excludes:       opts.Excludes,
		BackupStart:    time.Now(),
		Time:           time.Now(),
		ParentSnapshot: parentSnapshot,
		ProgramVersion: "resticlib",
	}

	// Run archiver
	_, snapshotID, summary, err := arch.Snapshot(ctx, resolvedPaths, snapshotOpts)
	if err != nil {
		return "", fmt.Errorf("backup failed: %w", err)
	}

	r.logf("info", "Backup completed successfully, snapshot ID: %s", snapshotID.Str())
	if summary != nil {
		r.logf("info", "Processed %d files, %d bytes", 
			summary.Files.New+summary.Files.Changed+summary.Files.Unchanged, 
			summary.ProcessedBytes)
	}

	return SnapshotID(snapshotID.Str()), nil
}