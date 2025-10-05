package resticlib

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/restic"
	"github.com/restic/restic/internal/restorer"
	"github.com/restic/restic/internal/ui/progress"
	"github.com/restic/restic/internal/ui/restore"
)

// restoreProgressWrapper adapts our ProgressReporter to restorer progress interface
type restoreProgressPrinter struct {
	reporter ProgressReporter
}

func (p *restoreProgressPrinter) Update(progress restore.State, duration time.Duration) {
	// Update progress state
}

func (p *restoreProgressPrinter) Error(item string, err error) error {
	if p.reporter != nil {
		return p.reporter.Error(item, err)
	}
	return err
}

func (p *restoreProgressPrinter) CompleteItem(action restore.ItemAction, item string, size uint64) {
	if p.reporter != nil {
		p.reporter.Add(size)
	}
}

func (p *restoreProgressPrinter) Finish(progress restore.State, duration time.Duration) {
	if p.reporter != nil {
		p.reporter.Finish()
	}
}

// NewCounter implements progress.Printer
func (p *restoreProgressPrinter) NewCounter(description string) *progress.Counter {
	return nil // Not implemented for now
}

// NewCounterTerminalOnly implements progress.Printer
func (p *restoreProgressPrinter) NewCounterTerminalOnly(description string) *progress.Counter {
	return nil // Not implemented for now
}

// E implements progress.Printer (Error reporting)
func (p *restoreProgressPrinter) E(msg string, args ...interface{}) {
	// Error reporting - could log via reporter
}

// S implements progress.Printer (Important messages)
func (p *restoreProgressPrinter) S(msg string, args ...interface{}) {
	// Important messages
}

// PT implements progress.Printer (Terminal messages)
func (p *restoreProgressPrinter) PT(msg string, args ...interface{}) {
	// Terminal messages
}

// P implements progress.Printer (Normal messages)
func (p *restoreProgressPrinter) P(msg string, args ...interface{}) {
	// Normal messages
}

// V implements progress.Printer (Verbose messages)
func (p *restoreProgressPrinter) V(msg string, args ...interface{}) {
	// Verbose messages
}

// VV implements progress.Printer (Debug messages)
func (p *restoreProgressPrinter) VV(msg string, args ...interface{}) {
	// Debug messages
}

// Restore restores files from a snapshot
func (r *repositoryImpl) Restore(ctx context.Context, snapshotID SnapshotID, opts RestoreOptions) error {
	r.logf("info", "Starting restore from snapshot %s to %s", snapshotID, opts.TargetDir)

	// Parse snapshot ID
	id, err := restic.ParseID(string(snapshotID))
	if err != nil {
		return fmt.Errorf("invalid snapshot ID: %w", err)
	}

	// Load snapshot
	sn, err := data.LoadSnapshot(ctx, r.repo, id)
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Load index
	err = r.repo.LoadIndex(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to load index: %w", err)
	}

	// Set up progress reporting
	var progress *restore.Progress
	if opts.Progress != nil {
		printer := &restoreProgressPrinter{reporter: opts.Progress}
		progress = restore.NewProgress(printer, 0) // 0 means no automatic updates
	}

	// Create restorer options
	restorerOpts := restorer.Options{
		DryRun:    opts.DryRun,
		Progress:  progress,
		Overwrite: restorer.OverwriteAlways, // Default overwrite behavior
		Delete:    opts.Delete,
	}
	
	if opts.Overwrite {
		restorerOpts.Overwrite = restorer.OverwriteAlways
	} else {
		restorerOpts.Overwrite = restorer.OverwriteIfNewer
	}

	// Create restorer
	res := restorer.NewRestorer(r.repo, sn, restorerOpts)

	// Set up includes/excludes
	var includePatterns []string
	var excludePatterns []string

	if len(opts.Includes) > 0 {
		includePatterns = opts.Includes
	}
	if len(opts.Excludes) > 0 {
		excludePatterns = opts.Excludes
	}

	// Set up selection function
	selectFilter := func(item string, isDir bool) (selectedForRestore bool, childMayBeSelected bool) {
		// Apply includes first (if any)
		if len(includePatterns) > 0 {
			matched := false
			for _, pattern := range includePatterns {
				if matched, _ = filepath.Match(pattern, item); matched {
					break
				}
				if matched, _ = filepath.Match(pattern, filepath.Base(item)); matched {
					break
				}
			}
			if !matched {
				return false, true // Don't restore this item, but children might match
			}
		}

		// Apply excludes
		for _, pattern := range excludePatterns {
			if matched, _ := filepath.Match(pattern, item); matched {
				return false, false // Don't restore this item or children
			}
			if matched, _ := filepath.Match(pattern, filepath.Base(item)); matched {
				return false, false // Don't restore this item or children
			}
		}

		return true, true
	}

	if len(includePatterns) > 0 || len(excludePatterns) > 0 {
		res.SelectFilter = selectFilter
	}

	// Perform restore
	filesRestored, err := res.RestoreTo(ctx, opts.TargetDir)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	r.logf("info", "Restored %d files", filesRestored)

	r.logf("info", "Restore completed successfully to %s", opts.TargetDir)
	return nil
}