package main

/*
#include <stdlib.h>
#include <string.h>
*/
import "C"

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/restic/restic/pkg/resticlib"
)

// Error codes
const (
	RESTIC_OK                     = 0
	RESTIC_ERROR_INVALID_PARAMS   = -1
	RESTIC_ERROR_REPO_NOT_FOUND   = -2
	RESTIC_ERROR_INVALID_PASSWORD = -3
	RESTIC_ERROR_BACKUP_FAILED    = -4
	RESTIC_ERROR_RESTORE_FAILED   = -5
	RESTIC_ERROR_UNKNOWN          = -99
)

// ResticRepo is an opaque pointer to a repository instance
type ResticRepo uintptr

// Global repository storage (in real implementation, you'd use a proper registry)
var repositories = make(map[ResticRepo]resticlib.Repository)
var nextRepoID ResticRepo = 1

// restic_init initializes a new repository
//
//export restic_init
func restic_init(repo_url *C.char, backend *C.char, password *C.char, access_key *C.char, secret_key *C.char, parallelism C.int) C.int {
	if repo_url == nil || backend == nil || password == nil {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	cfg := resticlib.Config{
		RepoURL:     C.GoString(repo_url),
		Backend:     resticlib.BackendKind(C.GoString(backend)),
		Password:    []byte(C.GoString(password)),
		Parallelism: int(parallelism),
	}

	if access_key != nil && secret_key != nil {
		cfg.Credentials = &resticlib.Credentials{
			AccessKey: C.GoString(access_key),
			SecretKey: C.GoString(secret_key),
		}
	}

	repo, err := resticlib.Init(ctx, cfg)
	if err != nil {
		return RESTIC_ERROR_REPO_NOT_FOUND
	}

	repoID := nextRepoID
	nextRepoID++
	repositories[repoID] = repo

	return C.int(repoID)
}

// restic_open opens an existing repository
//
//export restic_open
func restic_open(repo_url *C.char, backend *C.char, password *C.char, access_key *C.char, secret_key *C.char, parallelism C.int) C.int {
	if repo_url == nil || backend == nil || password == nil {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	cfg := resticlib.Config{
		RepoURL:     C.GoString(repo_url),
		Backend:     resticlib.BackendKind(C.GoString(backend)),
		Password:    []byte(C.GoString(password)),
		Parallelism: int(parallelism),
	}

	if access_key != nil && secret_key != nil {
		cfg.Credentials = &resticlib.Credentials{
			AccessKey: C.GoString(access_key),
			SecretKey: C.GoString(secret_key),
		}
	}

	repo, err := resticlib.Open(ctx, cfg)
	if err != nil {
		return RESTIC_ERROR_INVALID_PASSWORD
	}

	repoID := nextRepoID
	nextRepoID++
	repositories[repoID] = repo

	return C.int(repoID)
}

// restic_backup creates a backup and returns snapshot ID as string
//
//export restic_backup
func restic_backup(repo_id C.int, paths **C.char, paths_count C.int, tags **C.char, tags_count C.int, snapshot_id_out **C.char) C.int {
	repo, exists := repositories[ResticRepo(repo_id)]
	if !exists {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	if paths == nil || paths_count <= 0 {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	// Convert C arrays to Go slices
	pathSlice := make([]string, int(paths_count))
	cPaths := (*[1 << 30]*C.char)(unsafe.Pointer(paths))[:paths_count:paths_count]
	for i, cPath := range cPaths {
		pathSlice[i] = C.GoString(cPath)
	}

	var tagSlice []string
	if tags_count > 0 && tags != nil {
		tagSlice = make([]string, int(tags_count))
		cTags := (*[1 << 30]*C.char)(unsafe.Pointer(tags))[:tags_count:tags_count]
		for i, cTag := range cTags {
			tagSlice[i] = C.GoString(cTag)
		}
	}

	backupOpts := resticlib.BackupOptions{
		Paths: pathSlice,
		Tags:  tagSlice,
	}

	snapshotID, err := repo.Backup(ctx, backupOpts)
	if err != nil {
		return RESTIC_ERROR_BACKUP_FAILED
	}

	*snapshot_id_out = C.CString(string(snapshotID))
	return RESTIC_OK
}

// restic_restore restores a snapshot to target directory
//
//export restic_restore
func restic_restore(repo_id C.int, snapshot_id *C.char, target_dir *C.char) C.int {
	repo, exists := repositories[ResticRepo(repo_id)]
	if !exists {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	if snapshot_id == nil || target_dir == nil {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	restoreOpts := resticlib.RestoreOptions{
		TargetDir: C.GoString(target_dir),
		Overwrite: true,
	}

	err := repo.Restore(ctx, resticlib.SnapshotID(C.GoString(snapshot_id)), restoreOpts)
	if err != nil {
		return RESTIC_ERROR_RESTORE_FAILED
	}

	return RESTIC_OK
}

// restic_list_snapshots lists all snapshots in the repository
//
//export restic_list_snapshots
func restic_list_snapshots(repo_id C.int, ids_out ***C.char, times_out ***C.char, hostnames_out ***C.char, count_out *C.int) C.int {
	repo, exists := repositories[ResticRepo(repo_id)]
	if !exists {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	snapshots, err := repo.Snapshots(ctx, resticlib.SnapshotFilter{})
	if err != nil {
		return RESTIC_ERROR_UNKNOWN
	}

	if len(snapshots) == 0 {
		*count_out = 0
		return RESTIC_OK
	}

	// Allocate arrays for snapshot data
	count := len(snapshots)
	idsArray := C.malloc(C.size_t(count) * C.size_t(unsafe.Sizeof(uintptr(0))))
	timesArray := C.malloc(C.size_t(count) * C.size_t(unsafe.Sizeof(uintptr(0))))
	hostnamesArray := C.malloc(C.size_t(count) * C.size_t(unsafe.Sizeof(uintptr(0))))

	cIds := (*[1 << 30]*C.char)(idsArray)
	cTimes := (*[1 << 30]*C.char)(timesArray)
	cHostnames := (*[1 << 30]*C.char)(hostnamesArray)

	for i, snapshot := range snapshots {
		cIds[i] = C.CString(string(snapshot.ID))
		cTimes[i] = C.CString(snapshot.Time)
		cHostnames[i] = C.CString(snapshot.Hostname)
	}

	*ids_out = (**C.char)(idsArray)
	*times_out = (**C.char)(timesArray)
	*hostnames_out = (**C.char)(hostnamesArray)
	*count_out = C.int(count)
	return RESTIC_OK
}

// restic_check performs repository integrity check
//
//export restic_check
func restic_check(repo_id C.int, errors_out *C.int) C.int {
	repo, exists := repositories[ResticRepo(repo_id)]
	if !exists {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	ctx := context.Background()

	report, err := repo.Check(ctx, resticlib.CheckDepthDefault)
	if err != nil {
		return RESTIC_ERROR_UNKNOWN
	}

	*errors_out = C.int(len(report.Errors))
	return RESTIC_OK
}

// restic_close closes a repository and frees resources
//
//export restic_close
func restic_close(repo_id C.int) C.int {
	repo, exists := repositories[ResticRepo(repo_id)]
	if !exists {
		return RESTIC_ERROR_INVALID_PARAMS
	}

	repo.Close()
	delete(repositories, ResticRepo(repo_id))
	return RESTIC_OK
}

// restic_free_string frees a string returned by the library
//
//export restic_free_string
func restic_free_string(str *C.char) {
	if str != nil {
		C.free(unsafe.Pointer(str))
	}
}

// restic_free_snapshot_arrays frees arrays returned by restic_list_snapshots
//
//export restic_free_snapshot_arrays
func restic_free_snapshot_arrays(ids **C.char, times **C.char, hostnames **C.char, count C.int) {
	if ids != nil {
		cIds := (*[1 << 30]*C.char)(unsafe.Pointer(ids))
		for i := 0; i < int(count); i++ {
			if cIds[i] != nil {
				C.free(unsafe.Pointer(cIds[i]))
			}
		}
		C.free(unsafe.Pointer(ids))
	}

	if times != nil {
		cTimes := (*[1 << 30]*C.char)(unsafe.Pointer(times))
		for i := 0; i < int(count); i++ {
			if cTimes[i] != nil {
				C.free(unsafe.Pointer(cTimes[i]))
			}
		}
		C.free(unsafe.Pointer(times))
	}

	if hostnames != nil {
		cHostnames := (*[1 << 30]*C.char)(unsafe.Pointer(hostnames))
		for i := 0; i < int(count); i++ {
			if cHostnames[i] != nil {
				C.free(unsafe.Pointer(cHostnames[i]))
			}
		}
		C.free(unsafe.Pointer(hostnames))
	}
}

// restic_get_version returns the library version
//
//export restic_get_version
func restic_get_version() *C.char {
	return C.CString("resticlib-v0.1.0")
}

// restic_get_error_message returns a human-readable error message for an error code
//
//export restic_get_error_message
func restic_get_error_message(error_code C.int) *C.char {
	switch error_code {
	case RESTIC_OK:
		return C.CString("Success")
	case RESTIC_ERROR_INVALID_PARAMS:
		return C.CString("Invalid parameters")
	case RESTIC_ERROR_REPO_NOT_FOUND:
		return C.CString("Repository not found or invalid password")
	case RESTIC_ERROR_INVALID_PASSWORD:
		return C.CString("Invalid password")
	case RESTIC_ERROR_BACKUP_FAILED:
		return C.CString("Backup operation failed")
	case RESTIC_ERROR_RESTORE_FAILED:
		return C.CString("Restore operation failed")
	default:
		return C.CString("Unknown error")
	}
}

// Keep the main function for CGO to work
func main() {
	// This function is required for CGO but won't be called when used as a library
	fmt.Println("resticlib C bridge")
}
