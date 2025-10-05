// Package main demonstrates how to build a simple CLI using resticlib.
// This shows how the existing restic CLI could be refactored to use the library.
//
// Example usage:
//   export RESTIC_REPOSITORY=/tmp/my-backup
//   export RESTIC_PASSWORD=mypassword
//   go run examples/cli_example.go init
//   go run examples/cli_example.go backup /home/user/documents
//   go run examples/cli_example.go snapshots
//   go run examples/cli_example.go restore <snapshot-id> /tmp/restore
//   go run examples/cli_example.go check
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/restic/restic/pkg/resticlib"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: cli_example <command> [args...]")
		fmt.Println("Commands: init, backup, snapshots, restore, check")
		os.Exit(1)
	}

	// Configuration from environment variables (like original CLI)
	repoURL := os.Getenv("RESTIC_REPOSITORY")
	if repoURL == "" {
		log.Fatal("RESTIC_REPOSITORY environment variable not set")
	}

	password := os.Getenv("RESTIC_PASSWORD")
	if password == "" {
		log.Fatal("RESTIC_PASSWORD environment variable not set")
	}

	config := resticlib.Config{
		RepoURL:     repoURL,
		Backend:     detectBackend(repoURL),
		Password:    []byte(password),
		Parallelism: 4,
		Logger:      &resticlib.DefaultLogger{Writer: os.Stderr},
	}

	ctx := context.Background()
	command := os.Args[1]

	switch command {
	case "init":
		handleInit(ctx, config)
	case "backup":
		handleBackup(ctx, config, os.Args[2:])
	case "snapshots":
		handleSnapshots(ctx, config)
	case "restore":
		handleRestore(ctx, config, os.Args[2:])
	case "check":
		handleCheck(ctx, config)
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func detectBackend(repoURL string) resticlib.BackendKind {
	if len(repoURL) > 0 {
		switch {
		case repoURL[0] == '/' || (len(repoURL) > 1 && repoURL[1] == ':'):
			return resticlib.BackendLocal
		}
	}

	// Simple detection based on prefix
	switch {
	case len(repoURL) >= 3 && repoURL[:3] == "s3:":
		return resticlib.BackendS3
	case len(repoURL) >= 6 && repoURL[:6] == "azure:":
		return resticlib.BackendAzure
	case len(repoURL) >= 3 && repoURL[:3] == "gs:":
		return resticlib.BackendGCS
	case len(repoURL) >= 3 && repoURL[:3] == "b2:":
		return resticlib.BackendB2
	case len(repoURL) >= 5 && repoURL[:5] == "sftp:":
		return resticlib.BackendSFTP
	case len(repoURL) >= 6 && repoURL[:6] == "swift:":
		return resticlib.BackendSwift
	case len(repoURL) >= 5 && repoURL[:5] == "rest:":
		return resticlib.BackendRest
	default:
		return resticlib.BackendLocal
	}
}

func handleInit(ctx context.Context, config resticlib.Config) {
	fmt.Println("Initializing repository...")
	repo, err := resticlib.Init(ctx, config)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()
	fmt.Println("Repository initialized successfully")
}

func handleBackup(ctx context.Context, config resticlib.Config, args []string) {
	if len(args) == 0 {
		log.Fatal("No paths specified for backup")
	}

	repo, err := resticlib.Open(ctx, config)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}
	defer repo.Close()

	opts := resticlib.BackupOptions{
		Paths: args,
		Tags:  []string{"cli-example"},
	}

	fmt.Printf("Creating backup of: %v\n", args)
	snapshotID, err := repo.Backup(ctx, opts)
	if err != nil {
		log.Fatalf("Backup failed: %v", err)
	}
	fmt.Printf("Snapshot saved as: %s\n", snapshotID)
}

func handleSnapshots(ctx context.Context, config resticlib.Config) {
	repo, err := resticlib.Open(ctx, config)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}
	defer repo.Close()

	snapshots, err := repo.Snapshots(ctx, resticlib.SnapshotFilter{})
	if err != nil {
		log.Fatalf("Failed to list snapshots: %v", err)
	}

	fmt.Printf("ID        | Time                 | Host       | Tags       | Paths\n")
	fmt.Printf("----------|----------------------|------------|------------|----------\n")
	for _, sn := range snapshots {
		fmt.Printf("%-9s | %-20s | %-10s | %-10v | %v\n",
			sn.ID.String()[:8]+"...",
			sn.Time,
			sn.Hostname,
			sn.Tags,
			sn.Paths)
	}
}

func handleRestore(ctx context.Context, config resticlib.Config, args []string) {
	if len(args) < 2 {
		log.Fatal("Usage: restore <snapshot-id> <target-dir>")
	}

	repo, err := resticlib.Open(ctx, config)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}
	defer repo.Close()

	snapshotID := resticlib.SnapshotID(args[0])
	targetDir := args[1]

	opts := resticlib.RestoreOptions{
		TargetDir: targetDir,
		Overwrite: true,
	}

	fmt.Printf("Restoring snapshot %s to %s\n", snapshotID, targetDir)
	err = repo.Restore(ctx, snapshotID, opts)
	if err != nil {
		log.Fatalf("Restore failed: %v", err)
	}
	fmt.Println("Restore completed successfully")
}

func handleCheck(ctx context.Context, config resticlib.Config) {
	repo, err := resticlib.Open(ctx, config)
	if err != nil {
		log.Fatalf("Failed to open repository: %v", err)
	}
	defer repo.Close()

	fmt.Println("Checking repository integrity...")
	report, err := repo.Check(ctx, resticlib.CheckDepthDefault)
	if err != nil {
		log.Fatalf("Check failed: %v", err)
	}

	if report.Success {
		fmt.Println("Repository integrity check passed")
	} else {
		fmt.Printf("Repository integrity check failed:\n")
		for _, errMsg := range report.Errors {
			fmt.Printf("  ERROR: %s\n", errMsg)
		}
		for _, warning := range report.Warnings {
			fmt.Printf("  WARNING: %s\n", warning)
		}
		os.Exit(1)
	}
}