package resticlib_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/restic/restic/pkg/resticlib"
)

// This example shows how to use the resticlib API to initialize a repository,
// create a backup, list snapshots, and perform other operations.
func Example() {
	// Create a temporary directory for the repository
	tempDir := "/tmp/restic-example"
	_ = os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	ctx := context.Background()

	// Configure the repository
	config := resticlib.Config{
		RepoURL:     fmt.Sprintf("local:%s", tempDir),
		Backend:     resticlib.BackendLocal,
		Password:    []byte("mysecretpassword"),
		Parallelism: 4,
		Logger:      &resticlib.DefaultLogger{Writer: os.Stdout},
	}

	// Initialize a new repository
	repo, err := resticlib.Init(ctx, config)
	if err != nil {
		fmt.Printf("Failed to initialize repository: %v\n", err)
		return
	}
	defer repo.Close()

	fmt.Println("Repository initialized successfully")

	// Create a temporary file to backup
	testDir := "/tmp/restic-test-data"
	_ = os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("Hello, restic library!"), 0644)
	if err != nil {
		fmt.Printf("Failed to create test file: %v\n", err)
		return
	}

	// Create a backup
	backupOpts := resticlib.BackupOptions{
		Paths: []string{testDir},
		Tags:  []string{"example", "test"},
	}

	snapshotID, err := repo.Backup(ctx, backupOpts)
	if err != nil {
		fmt.Printf("Backup failed: %v\n", err)
		return
	}

	fmt.Printf("Backup created with snapshot ID: %s\n", snapshotID)

	// List snapshots
	filter := resticlib.SnapshotFilter{
		Tags:  []string{"example"},
		Limit: 10,
	}

	snapshots, err := repo.Snapshots(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to list snapshots: %v\n", err)
		return
	}

	fmt.Printf("Found %d snapshots:\n", len(snapshots))
	for _, sn := range snapshots {
		fmt.Printf("  ID: %s, Time: %s, Paths: %v\n", sn.ID, sn.Time, sn.Paths)
	}

	// Check repository integrity
	checkReport, err := repo.Check(ctx, resticlib.CheckDepthDefault)
	if err != nil {
		fmt.Printf("Check failed: %v\n", err)
		return
	}

	if checkReport.Success {
		fmt.Println("Repository integrity check passed")
	} else {
		fmt.Printf("Repository integrity check failed with %d errors\n", len(checkReport.Errors))
	}

	// Output: Repository initialized successfully
	// Backup created with snapshot ID: <snapshot-id>
	// Found 1 snapshots:
	//   ID: <snapshot-id>, Time: <timestamp>, Paths: [/tmp/restic-test-data]
	// Repository integrity check passed
}