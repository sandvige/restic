package resticlib

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/restic/restic/internal/data"
	"github.com/restic/restic/internal/restic"
)

// Snapshots lists snapshots matching the filter
func (r *repositoryImpl) Snapshots(ctx context.Context, filter SnapshotFilter) ([]Snapshot, error) {
	r.logf("debug", "Listing snapshots with filter: %+v", filter)

	// Load all snapshots from repository
	var allSnapshots data.Snapshots
	err := r.repo.List(ctx, restic.SnapshotFile, func(id restic.ID, size int64) error {
		sn, err := data.LoadSnapshot(ctx, r.repo, id)
		if err != nil {
			r.logf("warn", "Failed to load snapshot %s: %v", id.Str(), err)
			return nil // Continue with other snapshots
		}
		allSnapshots = append(allSnapshots, sn)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Filter snapshots based on criteria
	var filteredSnapshots data.Snapshots
	for _, sn := range allSnapshots {
		if r.matchesFilter(sn, filter) {
			filteredSnapshots = append(filteredSnapshots, sn)
		}
	}

	// Sort by time (newest first)
	sort.Slice(filteredSnapshots, func(i, j int) bool {
		return filteredSnapshots[i].Time.After(filteredSnapshots[j].Time)
	})

	// Apply limit if specified
	if filter.Limit > 0 && len(filteredSnapshots) > filter.Limit {
		filteredSnapshots = filteredSnapshots[:filter.Limit]
	}

	// Convert to library types
	result := make([]Snapshot, len(filteredSnapshots))
	for i, sn := range filteredSnapshots {
		result[i] = r.convertSnapshot(sn)
	}

	r.logf("info", "Found %d snapshots matching criteria", len(result))
	return result, nil
}

// matchesFilter checks if a snapshot matches the given filter criteria
func (r *repositoryImpl) matchesFilter(sn *data.Snapshot, filter SnapshotFilter) bool {
	// Check hosts
	if len(filter.Hosts) > 0 {
		found := false
		for _, host := range filter.Hosts {
			if sn.Hostname == host {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check paths
	if len(filter.Paths) > 0 {
		found := false
		for _, filterPath := range filter.Paths {
			for _, snPath := range sn.Paths {
				if strings.Contains(snPath, filterPath) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check tags
	if len(filter.Tags) > 0 {
		found := false
		for _, filterTag := range filter.Tags {
			for _, snTag := range sn.Tags {
				if snTag == filterTag {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if filter.Since != nil {
		since, err := time.Parse(time.RFC3339, *filter.Since)
		if err == nil && sn.Time.Before(since) {
			return false
		}
	}

	if filter.Until != nil {
		until, err := time.Parse(time.RFC3339, *filter.Until)
		if err == nil && sn.Time.After(until) {
			return false
		}
	}

	return true
}

// convertSnapshot converts an internal snapshot to library type
func (r *repositoryImpl) convertSnapshot(sn *data.Snapshot) Snapshot {
	result := Snapshot{
		ID:       SnapshotID(sn.ID().Str()),
		Time:     sn.Time.Format(time.RFC3339),
		Tree:     sn.Tree.Str(),
		Paths:    sn.Paths,
		Hostname: sn.Hostname,
		Username: sn.Username,
		Tags:     sn.Tags,
	}

	if sn.Parent != nil {
		parent := sn.Parent.Str()
		result.Parent = &parent
	}

	return result
}