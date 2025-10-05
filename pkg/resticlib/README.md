# resticlib - Go Library for Restic Backup Operations

`resticlib` is a Go library that exposes restic's core backup functionality through a clean, programmatic API. This library allows you to integrate restic's backup, restore, and repository management capabilities directly into your Go applications without needing to execute the restic binary.

## Features

- **Repository Operations**: Initialize and open restic repositories
- **Backup**: Create incremental backups with full control over paths, excludes, and metadata
- **Restore**: Restore snapshots to specified locations with flexible filtering
- **Snapshot Management**: List, filter, and manage snapshots programmatically
- **Retention Policies**: Apply forget policies to automatically remove old snapshots
- **Repository Maintenance**: Prune unused data, check integrity, and unlock stale locks
- **Backend Support**: Full support for all restic backends (local, S3, Azure, GCS, B2, SFTP, Swift, REST)
- **Context Support**: Proper context handling for cancellation and timeouts
- **Progress Reporting**: Pluggable progress reporting interfaces
- **Structured Logging**: Configurable logging with clean interfaces

## Installation

```bash
go get github.com/restic/restic/pkg/resticlib
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/restic/restic/pkg/resticlib"
)

func main() {
    ctx := context.Background()

    // Configure repository
    config := resticlib.Config{
        RepoURL:  "s3:s3.amazonaws.com/my-backup-bucket",
        Backend:  resticlib.BackendS3,
        Password: []byte("mysecretpassword"),
        Credentials: &resticlib.Credentials{
            AccessKey: "AKIA...",
            SecretKey: "...",
        },
        Parallelism: 4,
        Logger: &resticlib.DefaultLogger{Writer: os.Stderr},
    }

    // Initialize or open repository
    repo, err := resticlib.Open(ctx, config)
    if err != nil {
        // If repository doesn't exist, initialize it
        repo, err = resticlib.Init(ctx, config)
        if err != nil {
            panic(err)
        }
    }
    defer repo.Close()

    // Create a backup
    snapshotID, err := repo.Backup(ctx, resticlib.BackupOptions{
        Paths: []string{"/home/user/documents"},
        Tags:  []string{"documents", "daily"},
        Excludes: []string{"*.tmp", "*.log"},
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Backup created: %s\n", snapshotID)

    // List snapshots
    snapshots, err := repo.Snapshots(ctx, resticlib.SnapshotFilter{
        Tags: []string{"documents"},
        Limit: 10,
    })
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d snapshots\n", len(snapshots))
}
```

## API Reference

### Core Types

#### Repository Interface
```go
type Repository interface {
    Backup(ctx context.Context, opts BackupOptions) (SnapshotID, error)
    Restore(ctx context.Context, snapshotID SnapshotID, opts RestoreOptions) error
    Snapshots(ctx context.Context, filter SnapshotFilter) ([]Snapshot, error)
    Forget(ctx context.Context, policy ForgetPolicy) ([]SnapshotID, error)
    Prune(ctx context.Context, opts PruneOptions) (PruneReport, error)
    Check(ctx context.Context, depth CheckDepth) (CheckReport, error)
    Unlock(ctx context.Context) error
    Close() error
}
```

#### Configuration
```go
type Config struct {
    RepoURL      string         // Repository location (e.g., "s3:bucket/path")
    Backend      BackendKind    // Storage backend type
    Credentials  *Credentials   // Authentication credentials
    Password     []byte         // Repository encryption password
    CACertsPEM   []byte         // Custom CA certificates
    Parallelism  int            // Number of concurrent operations
    TempDir      string         // Temporary directory for operations
    Logger       Logger         // Logging interface
}
```

### Backend Support

The library supports all restic backends:

- **Local**: `local:/path/to/repo` or `/path/to/repo`
- **S3**: `s3:s3.amazonaws.com/bucket/path`
- **Azure**: `azure:container/path`
- **Google Cloud**: `gs:bucket/path`
- **Backblaze B2**: `b2:bucket/path`
- **SFTP**: `sftp:user@host:/path`
- **Swift**: `swift:container/path`
- **REST**: `rest:http://host:port/`

### Operations

#### Initialize Repository
```go
repo, err := resticlib.Init(ctx, resticlib.Config{
    RepoURL:  "local:/tmp/backup",
    Backend:  resticlib.BackendLocal,
    Password: []byte("password123"),
})
```

#### Open Existing Repository
```go
repo, err := resticlib.Open(ctx, config)
```

#### Create Backup
```go
snapshotID, err := repo.Backup(ctx, resticlib.BackupOptions{
    Paths:    []string{"/home/user"},
    Tags:     []string{"home", "user-data"},
    Excludes: []string{"*.cache", "*/tmp/*"},
    ParentID: &previousSnapshotID, // Optional incremental backup
})
```

#### Restore Data
```go
err := repo.Restore(ctx, snapshotID, resticlib.RestoreOptions{
    TargetDir: "/restore/location",
    Includes:  []string{"documents/*"},
    Overwrite: true,
})
```

#### List Snapshots
```go
snapshots, err := repo.Snapshots(ctx, resticlib.SnapshotFilter{
    Hosts: []string{"laptop", "server"},
    Tags:  []string{"important"},
    Since: &"2024-01-01T00:00:00Z",
    Limit: 20,
})
```

#### Apply Retention Policy
```go
removedIDs, err := repo.Forget(ctx, resticlib.ForgetPolicy{
    KeepLast:    5,
    KeepDaily:   7,
    KeepWeekly:  4,
    KeepMonthly: 6,
    KeepYearly:  2,
    KeepWithin: &"30d",
})
```

#### Repository Maintenance
```go
// Check integrity
report, err := repo.Check(ctx, resticlib.CheckDepthDefault)

// Remove unused data
pruneReport, err := repo.Prune(ctx, resticlib.PruneOptions{
    DryRun: false,
})

// Remove stale locks
err := repo.Unlock(ctx)
```

### Progress Reporting

Implement custom progress reporting:

```go
type MyProgressReporter struct{}

func (p *MyProgressReporter) SetTotal(total uint64) {
    fmt.Printf("Total: %d bytes\n", total)
}

func (p *MyProgressReporter) Add(delta uint64) {
    fmt.Printf("Progress: +%d bytes\n", delta)
}

func (p *MyProgressReporter) Error(item string, err error) error {
    fmt.Printf("Error with %s: %v\n", item, err)
    return nil // Continue on error
}

func (p *MyProgressReporter) Finish() {
    fmt.Println("Operation completed")
}

// Use in operations
backupOpts.Progress = &MyProgressReporter{}
```

### Logging

Implement custom logging:

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...interface{}) {
    log.Printf("[DEBUG] "+msg, args...)
}

func (l *MyLogger) Info(msg string, args ...interface{}) {
    log.Printf("[INFO] "+msg, args...)
}

func (l *MyLogger) Warn(msg string, args ...interface{}) {
    log.Printf("[WARN] "+msg, args...)
}

func (l *MyLogger) Error(msg string, args ...interface{}) {
    log.Printf("[ERROR] "+msg, args...)
}

config.Logger = &MyLogger{}
```

## Repository Compatibility

The library maintains full compatibility with repositories created by the restic CLI:

- **Read/Write Compatibility**: Repositories created with the library can be used with the CLI and vice versa
- **Format Compatibility**: Uses the same data formats, encryption, and chunking algorithms
- **Metadata Compatibility**: Snapshots created by the library are fully compatible with CLI tools

## Thread Safety

- **Repository instances are NOT thread-safe** and should not be shared between goroutines
- **Multiple repositories** can be safely used concurrently from different goroutines
- **Backend operations** are internally synchronized where appropriate

## Security Considerations

- **Passwords**: Never logged or exposed in error messages
- **Memory**: Sensitive data is cleared when possible
- **Credentials**: Support for various authentication methods per backend
- **Encryption**: Uses restic's proven cryptographic implementations

## Error Handling

The library uses typed errors where appropriate:

```go
if errors.Is(err, resticlib.ErrRepositoryNotFound) {
    // Handle missing repository
}

if errors.Is(err, resticlib.ErrInvalidPassword) {
    // Handle authentication failure
}
```

## Migration from CLI

The library provides a straightforward migration path from CLI usage:

| CLI Command | Library Method |
|-------------|----------------|
| `restic init` | `resticlib.Init()` |
| `restic backup` | `repo.Backup()` |
| `restic restore` | `repo.Restore()` |
| `restic snapshots` | `repo.Snapshots()` |
| `restic forget` | `repo.Forget()` |
| `restic prune` | `repo.Prune()` |
| `restic check` | `repo.Check()` |
| `restic unlock` | `repo.Unlock()` |

## Examples

See the `cli_example.go` file for a complete example of how to build a CLI application using the library, demonstrating how the existing restic CLI could be refactored to use this library.

## License

This library is part of the restic project and is released under the same license as restic (BSD 2-Clause License).