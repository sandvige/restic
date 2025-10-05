package resticlib

import (
	"context"
	"fmt"

	"github.com/restic/restic/internal/repository"
)

// Check verifies repository integrity
func (r *repositoryImpl) Check(ctx context.Context, depth CheckDepth) (CheckReport, error) {
	r.logf("info", "Starting integrity check (depth: %s)", depth)

	report := CheckReport{
		Errors:   []string{},
		Warnings: []string{},
		Success:  true,
	}

	// Load index
	err := r.repo.LoadIndex(ctx, nil)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("failed to load index: %v", err))
		report.Success = false
		return report, err
	}

	// Create checker
	checker := r.repo.Checker()

	// Load checker index
	hints, errs := checker.LoadIndex(ctx, nil)

	// Process hints (warnings)
	for _, hint := range hints {
		report.Warnings = append(report.Warnings, hint.Error())
	}

	// Process errors
	for _, err := range errs {
		report.Errors = append(report.Errors, err.Error())
		report.Success = false
	}

	if len(errs) > 0 {
		r.logf("error", "Index check failed with %d errors", len(errs))
		return report, fmt.Errorf("index check failed")
	}

	// Check packs
	r.logf("debug", "Checking pack files")
	errChan := make(chan error, 100)
	go func() {
		checker.Packs(ctx, errChan)
		// Note: Packs() closes the channel itself
	}()

	packErrors := 0
	for err := range errChan {
		report.Errors = append(report.Errors, fmt.Sprintf("pack error: %v", err))
		report.Success = false
		packErrors++
	}

	if packErrors > 0 {
		r.logf("error", "Pack check failed with %d errors", packErrors)
	}

	// For read-data depth, actually read and verify data
	if depth == CheckDepthReadData {
		r.logf("debug", "Reading and verifying pack data")

		dataErrChan := make(chan error, 100)
		go func() {
			checker.ReadData(ctx, dataErrChan)
			// Note: ReadData() -> ReadPacks() closes the channel itself
		}()

		dataErrors := 0
		for err := range dataErrChan {
			report.Errors = append(report.Errors, fmt.Sprintf("data error: %v", err))
			report.Success = false
			dataErrors++
		}

		if dataErrors > 0 {
			r.logf("error", "Data verification failed with %d errors", dataErrors)
		}
	}

	if report.Success {
		r.logf("info", "Integrity check completed successfully")
	} else {
		r.logf("error", "Integrity check found %d errors and %d warnings",
			len(report.Errors), len(report.Warnings))
	}

	return report, nil
}

// Unlock removes stale locks from repository
func (r *repositoryImpl) Unlock(ctx context.Context) error {
	r.logf("info", "Removing stale locks from repository")

	// Use the internal RemoveStaleLocks function which handles the proper lock removal
	removedCount, err := repository.RemoveStaleLocks(ctx, r.repo)
	if err != nil {
		return fmt.Errorf("failed to remove stale locks: %w", err)
	}

	if removedCount > 0 {
		r.logf("info", "Removed %d stale locks", removedCount)
	} else {
		r.logf("info", "No stale locks found")
	}

	return nil
}
