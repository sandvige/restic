package resticlib

import (
	"context"
	"fmt"

	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
)

// Forget removes snapshots according to policy
func (r *repositoryImpl) Forget(ctx context.Context, policy ForgetPolicy) ([]SnapshotID, error) {
	if policy.Empty() {
		return nil, errors.New("forget policy is empty")
	}

	r.logf("info", "Applying forget policy: %+v", policy)

	// Load all snapshots
	var allSnapshots data.Snapshots
	err := r.repo.List(ctx, restic.SnapshotFile, func(id restic.ID, size int64) error {
		sn, err := data.LoadSnapshot(ctx, r.repo, id)
		if err != nil {
			r.logf("warn", "Failed to load snapshot %s: %v", id.Str(), err)
			return nil
		}
		allSnapshots = append(allSnapshots, sn)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Group snapshots by hostname and paths
	groupBy := data.SnapshotGroupByOptions{Host: true, Path: true}
	groups, _, err := data.GroupSnapshots(allSnapshots, groupBy)
	if err != nil {
		return nil, fmt.Errorf("failed to group snapshots: %w", err)
	}

	var removedIDs []SnapshotID

	for _, group := range groups {
		// Convert policy to internal format
		internalPolicy := data.ExpirePolicy{
			Last:    policy.KeepLast,
			Hourly:  policy.KeepHourly,
			Daily:   policy.KeepDaily,
			Weekly:  policy.KeepWeekly,
			Monthly: policy.KeepMonthly,
			Yearly:  policy.KeepYearly,
		}

		// Convert tags to TagList
		if len(policy.KeepTags) > 0 {
			tagList := make(data.TagList, len(policy.KeepTags))
			for i, tag := range policy.KeepTags {
				tagList[i] = tag
			}
			internalPolicy.Tags = []data.TagList{tagList}
		}

		if policy.KeepWithin != nil {
			within, err := data.ParseDuration(*policy.KeepWithin)
			if err != nil {
				return nil, fmt.Errorf("invalid keep-within duration %q: %w", *policy.KeepWithin, err)
			}
			internalPolicy.Within = within
		}

		// Apply policy to group
		keep, remove, _ := data.ApplyPolicy(group, internalPolicy)

		// Safety check: don't remove all snapshots
		if len(keep) == 0 && len(remove) > 0 {
			r.logf("warn", "Refusing to delete last snapshot of group")
			continue
		}

		// Remove snapshots
		for _, sn := range remove {
			err := r.repo.RemoveUnpacked(ctx, restic.WriteableSnapshotFile, *sn.ID())
			if err != nil {
				r.logf("error", "Failed to remove snapshot %s: %v", sn.ID().Str(), err)
				continue
			}
			removedIDs = append(removedIDs, SnapshotID(sn.ID().String()))
			r.logf("info", "Removed snapshot %s", sn.ID().String())
		}
	}

	r.logf("info", "Forget completed, removed %d snapshots", len(removedIDs))
	return removedIDs, nil
}

// Prune removes unused data from repository
func (r *repositoryImpl) Prune(ctx context.Context, opts PruneOptions) (PruneReport, error) {
	r.logf("info", "Starting prune operation (dry-run: %v)", opts.DryRun)

	// Load index
	err := r.repo.LoadIndex(ctx, nil)
	if err != nil {
		return PruneReport{}, fmt.Errorf("failed to load index: %w", err)
	}

	// Create repository wrapper for prune operations
	repoWrapper := &internalRepository{r.repo}

	// Perform prune - this is a simplified version
	// In a real implementation, we would use the internal prune logic
	stats, err := r.performPrune(ctx, repoWrapper, opts)
	if err != nil {
		return PruneReport{}, fmt.Errorf("prune failed: %w", err)
	}

	r.logf("info", "Prune completed: deleted %d packs, repacked %d packs",
		stats.PacksDeleted, stats.PacksRepacked)

	return stats, nil
}

// performPrune performs the actual prune operation
func (r *repositoryImpl) performPrune(ctx context.Context, repo *internalRepository, opts PruneOptions) (PruneReport, error) {
	// This is a simplified prune implementation
	// A full implementation would include the complex prune logic from internal/repository

	report := PruneReport{
		PacksDeleted:  0,
		PacksKept:     0,
		PacksRepacked: 0,
		BytesDeleted:  0,
		BytesRepacked: 0,
	}

	// Count existing packs
	packCount := 0
	err := r.repo.List(ctx, restic.PackFile, func(id restic.ID, size int64) error {
		packCount++
		report.PacksKept++
		return nil
	})
	if err != nil {
		return report, err
	}

	r.logf("info", "Found %d packs in repository", packCount)

	// For now, we don't actually perform pruning to avoid data loss
	// A full implementation would analyze pack usage and remove unused packs
	if !opts.DryRun {
		r.logf("warn", "Actual pruning not implemented yet - this is a dry-run")
	}

	return report, nil
}

// internalRepository wraps our repository for internal operations
type internalRepository struct {
	*repository.Repository
}
