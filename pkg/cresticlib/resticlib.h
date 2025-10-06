#ifndef RESTICLIB_H
#define RESTICLIB_H

#ifdef __cplusplus
extern "C" {
#endif

/* Error codes */
#define RESTIC_OK                    0
#define RESTIC_ERROR_INVALID_PARAMS -1
#define RESTIC_ERROR_REPO_NOT_FOUND -2
#define RESTIC_ERROR_INVALID_PASSWORD -3
#define RESTIC_ERROR_BACKUP_FAILED   -4
#define RESTIC_ERROR_RESTORE_FAILED  -5
#define RESTIC_ERROR_UNKNOWN        -99

/* Note: This interface uses simple parameters to avoid complex struct passing */

/**
 * Initialize a new repository
 * @param repo_url Repository URL (e.g., "/path/to/repo" or "s3:bucket/path")
 * @param backend Backend type: "local", "s3", "azure", "gcs", "b2", "sftp", "swift", "rest"
 * @param password Repository password
 * @param access_key Access key for cloud backends (optional, can be NULL)
 * @param secret_key Secret key for cloud backends (optional, can be NULL)
 * @param parallelism Number of parallel workers
 * @return Repository ID (>= 0) on success, error code (< 0) on failure
 */
extern int restic_init(char* repo_url, char* backend, char* password, char* access_key, char* secret_key, int parallelism);

/**
 * Open an existing repository
 * @param repo_url Repository URL
 * @param backend Backend type
 * @param password Repository password
 * @param access_key Access key for cloud backends (optional, can be NULL)
 * @param secret_key Secret key for cloud backends (optional, can be NULL)  
 * @param parallelism Number of parallel workers
 * @return Repository ID (>= 0) on success, error code (< 0) on failure
 */
extern int restic_open(char* repo_url, char* backend, char* password, char* access_key, char* secret_key, int parallelism);

/**
 * Create a backup
 * @param repo_id Repository ID from restic_init/restic_open
 * @param paths Array of paths to backup
 * @param paths_count Number of paths
 * @param tags Array of tags (optional, can be NULL)
 * @param tags_count Number of tags
 * @param snapshot_id_out Output parameter for snapshot ID (caller must free with restic_free_string)
 * @return RESTIC_OK on success, error code on failure
 */
extern int restic_backup(int repo_id, char** paths, int paths_count, char** tags, int tags_count, char** snapshot_id_out);

/**
 * Restore a snapshot to target directory
 * @param repo_id Repository ID
 * @param snapshot_id Snapshot ID to restore
 * @param target_dir Target directory for restoration
 * @return RESTIC_OK on success, error code on failure
 */
extern int restic_restore(int repo_id, char* snapshot_id, char* target_dir);

/**
 * List all snapshots in repository
 * @param repo_id Repository ID
 * @param ids_out Output parameter for snapshot IDs array (caller must free with restic_free_snapshot_arrays)
 * @param times_out Output parameter for snapshot times array (caller must free with restic_free_snapshot_arrays)
 * @param hostnames_out Output parameter for hostnames array (caller must free with restic_free_snapshot_arrays)
 * @param count_out Output parameter for number of snapshots
 * @return RESTIC_OK on success, error code on failure
 */
extern int restic_list_snapshots(int repo_id, char*** ids_out, char*** times_out, char*** hostnames_out, int* count_out);

/**
 * Perform repository integrity check
 * @param repo_id Repository ID
 * @param errors_out Output parameter for number of errors found
 * @return RESTIC_OK on success, error code on failure
 */
extern int restic_check(int repo_id, int* errors_out);

/**
 * Close repository and free resources
 * @param repo_id Repository ID
 * @return RESTIC_OK on success, error code on failure
 */
extern int restic_close(int repo_id);

/**
 * Free a string returned by the library
 * @param str String to free
 */
extern void restic_free_string(char* str);

/**
 * Free arrays returned by restic_list_snapshots
 * @param ids Snapshot IDs array
 * @param times Snapshot times array
 * @param hostnames Hostnames array
 * @param count Number of snapshots
 */
extern void restic_free_snapshot_arrays(char** ids, char** times, char** hostnames, int count);

/**
 * Get library version
 * @return Version string (caller must free with restic_free_string)
 */
extern char* restic_get_version(void);

/**
 * Get human-readable error message for error code
 * @param error_code Error code from library functions
 * @return Error message (caller must free with restic_free_string)
 */
extern char* restic_get_error_message(int error_code);

#ifdef __cplusplus
}
#endif

#endif /* RESTICLIB_H */