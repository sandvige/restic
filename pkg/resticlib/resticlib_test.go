package resticlib

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBasicAPI tests that the basic API functions compile and can be called
func TestBasicAPI(t *testing.T) {
	// Test Config creation
	config := Config{
		RepoURL:     "local:/tmp/test-repo",
		Backend:     BackendLocal,
		Password:    []byte("testpassword"),
		Parallelism: 2,
		Logger:      &DefaultLogger{Writer: &bytes.Buffer{}},
	}

	// Test that we can call the functions (they may fail, but should not panic)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test Init function signature
	_, err := Init(ctx, config)
	if err != nil {
		t.Logf("Init failed as expected (no valid repo): %v", err)
	}

	// Test Open function signature  
	_, err = Open(ctx, config)
	if err != nil {
		t.Logf("Open failed as expected (no repo): %v", err)
	}
}

// TestTypes tests that all the basic types can be created
func TestTypes(t *testing.T) {
	// Test SnapshotID
	id := SnapshotID("test123")
	if id.String() != "test123" {
		t.Errorf("SnapshotID.String() = %v, want %v", id.String(), "test123")
	}

	// Test BackupOptions
	backupOpts := BackupOptions{
		Paths:    []string{"/test"},
		Tags:     []string{"test"},
		Excludes: []string{"*.tmp"},
		DryRun:   true,
	}
	if len(backupOpts.Paths) != 1 {
		t.Errorf("BackupOptions.Paths length = %v, want 1", len(backupOpts.Paths))
	}

	// Test RestoreOptions
	restoreOpts := RestoreOptions{
		TargetDir: "/tmp/restore",
		Overwrite: true,
		DryRun:    true,
	}
	if restoreOpts.TargetDir != "/tmp/restore" {
		t.Errorf("RestoreOptions.TargetDir = %v, want /tmp/restore", restoreOpts.TargetDir)
	}

	// Test SnapshotFilter
	filter := SnapshotFilter{
		Hosts: []string{"localhost"},
		Paths: []string{"/home"},
		Tags:  []string{"daily"},
		Limit: 10,
	}
	if filter.Limit != 10 {
		t.Errorf("SnapshotFilter.Limit = %v, want 10", filter.Limit)
	}

	// Test ForgetPolicy
	policy := ForgetPolicy{
		KeepLast:   5,
		KeepDaily:  7,
		KeepWeekly: 4,
	}
	if policy.Empty() {
		t.Errorf("ForgetPolicy.Empty() = true, want false")
	}

	// Test empty policy
	emptyPolicy := ForgetPolicy{}
	if !emptyPolicy.Empty() {
		t.Errorf("Empty ForgetPolicy.Empty() = false, want true")
	}

	// Test CheckDepth constants
	depths := []CheckDepth{
		CheckDepthDefault,
		CheckDepthFull,
		CheckDepthReadData,
	}
	if len(depths) != 3 {
		t.Errorf("Expected 3 CheckDepth constants, got %v", len(depths))
	}
}

// TestLogger tests the DefaultLogger implementation
func TestLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := &DefaultLogger{Writer: buf}

	logger.Info("test message %s", "arg")
	output := buf.String()
	if output == "" {
		t.Error("Expected log output, got empty string")
	}
	if !bytes.Contains(buf.Bytes(), []byte("INFO")) {
		t.Errorf("Expected INFO in log output, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte("test message arg")) {
		t.Errorf("Expected message in log output, got: %s", output)
	}

	// Test other log levels
	buf.Reset()
	logger.Warn("warning")
	if !bytes.Contains(buf.Bytes(), []byte("WARN")) {
		t.Errorf("Expected WARN in output, got: %s", buf.String())
	}

	buf.Reset()
	logger.Error("error")
	if !bytes.Contains(buf.Bytes(), []byte("ERROR")) {
		t.Errorf("Expected ERROR in output, got: %s", buf.String())
	}

	// Debug should not produce output in DefaultLogger
	buf.Reset()
	logger.Debug("debug")
	if buf.Len() != 0 {
		t.Errorf("Expected no debug output, got: %s", buf.String())
	}
}

// TestBackendKindConstants tests backend constants
func TestBackendKindConstants(t *testing.T) {
	backends := []BackendKind{
		BackendLocal,
		BackendS3,
		BackendAzure,
		BackendGCS,
		BackendB2,
		BackendSFTP,
		BackendSwift,
		BackendRest,
	}

	expectedValues := []string{
		"local",
		"s3",
		"azure",
		"gcs",
		"b2",
		"sftp",
		"swift",
		"rest",
	}

	for i, backend := range backends {
		if string(backend) != expectedValues[i] {
			t.Errorf("Backend %d: got %v, want %v", i, backend, expectedValues[i])
		}
	}
}

// TestRealRepository tests with an actual temporary repository
// This demonstrates the library works end-to-end
func TestRealRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory for repository
	tempDir, err := os.MkdirTemp("", "resticlib-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	repoPath := filepath.Join(tempDir, "repo")
	
	config := Config{
		RepoURL:     "local:" + repoPath,
		Backend:     BackendLocal,
		Password:    []byte("testpassword123"),
		Parallelism: 2,
		Logger:      &DefaultLogger{Writer: os.Stderr},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test repository initialization
	repo, err := Init(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	t.Logf("Repository initialized successfully at %s", repoPath)

	// Test opening the same repository
	repo2, err := Open(ctx, config)
	if err != nil {
		t.Fatalf("Failed to open repository: %v", err)
	}
	defer repo2.Close()

	t.Log("Repository opened successfully")

	// Create test data
	testDataDir := filepath.Join(tempDir, "data")
	err = os.MkdirAll(testDataDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test data dir: %v", err)
	}

	testFile := filepath.Join(testDataDir, "test.txt")
	testContent := []byte("Hello resticlib test!")
	err = os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test backup
	backupOpts := BackupOptions{
		Paths: []string{testDataDir},
		Tags:  []string{"test", "integration"},
	}

	snapshotID, err := repo2.Backup(ctx, backupOpts)
	if err != nil {
		t.Fatalf("Backup failed: %v", err)
	}

	t.Logf("Backup created with snapshot ID: %s (length: %d)", snapshotID, len(string(snapshotID)))

	// Test listing snapshots
	snapshots, err := repo2.Snapshots(ctx, SnapshotFilter{})
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 1 {
		t.Fatalf("Expected 1 snapshot, got %d", len(snapshots))
	}

	if snapshots[0].ID != snapshotID {
		t.Errorf("Snapshot ID mismatch: got %v, want %v", snapshots[0].ID, snapshotID)
	}

	t.Logf("Found snapshot: %+v", snapshots[0])
	t.Logf("Snapshot ID from list: %s (length: %d)", snapshots[0].ID, len(string(snapshots[0].ID)))

	// Test restore
	restoreDir := filepath.Join(tempDir, "restore")
	err = os.MkdirAll(restoreDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create restore dir: %v", err)
	}

	restoreOpts := RestoreOptions{
		TargetDir: restoreDir,
		Overwrite: true,
	}

	err = repo2.Restore(ctx, snapshotID, restoreOpts)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// List contents of restore directory to see structure
	entries, err := os.ReadDir(restoreDir)
	if err != nil {
		t.Fatalf("Failed to read restore dir: %v", err)
	}
	
	t.Logf("Restore directory contents:")
	for _, entry := range entries {
		t.Logf("  %s (dir: %v)", entry.Name(), entry.IsDir())
	}

	// The file should be restored with the full path structure
	// testDataDir is like "/tmp/resticlib-test-773598265/data"
	// So we need to go: restoreDir + testDataDir + "test.txt"
	restoredFile := filepath.Join(restoreDir, testDataDir, "test.txt")
	restoredContent, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if !bytes.Equal(restoredContent, testContent) {
		t.Errorf("Restored content mismatch: got %q, want %q", restoredContent, testContent)
	}

	t.Log("Restore successful, content verified")

	// Test check
	checkReport, err := repo2.Check(ctx, CheckDepthDefault)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !checkReport.Success {
		t.Errorf("Repository check failed: %+v", checkReport)
	}

	t.Log("Repository check passed")
}