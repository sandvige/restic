#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "../pkg/cresticlib/resticlib.h"

void print_error(int error_code) {
    char* error_msg = restic_get_error_message(error_code);
    printf("Error: %s\n", error_msg);
    restic_free_string(error_msg);
}

int main() {
    printf("ResticLib C Example\n");
    printf("==================\n\n");
    
    // Get library version
    char* version = restic_get_version();
    printf("Library version: %s\n\n", version);
    restic_free_string(version);
    
    // Initialize a new repository
    printf("Initializing repository...\n");
    int repo_id = restic_init("/tmp/restic-test", "local", "testpassword", NULL, NULL, 2);
    if (repo_id < 0) {
        print_error(repo_id);
        return 1;
    }
    printf("Repository initialized with ID: %d\n\n", repo_id);
    
    // Create some test directories and files
    system("mkdir -p /tmp/test-backup/dir1 /tmp/test-backup/dir2");
    system("echo 'Hello World' > /tmp/test-backup/file1.txt");
    system("echo 'Test content' > /tmp/test-backup/dir1/file2.txt");
    system("echo 'More data' > /tmp/test-backup/dir2/file3.txt");
    
    // Create a backup
    printf("Creating backup...\n");
    char* paths[] = {"/tmp/test-backup"};
    char* tags[] = {"example", "test"};
    char* snapshot_id = NULL;
    
    int result = restic_backup(repo_id, paths, 1, tags, 2, &snapshot_id);
    if (result != RESTIC_OK) {
        print_error(result);
        restic_close(repo_id);
        return 1;
    }
    
    printf("Backup created with snapshot ID: %s\n\n", snapshot_id);
    
    // List snapshots
    printf("Listing snapshots...\n");
    char** ids = NULL;
    char** times = NULL;
    char** hostnames = NULL;
    int count = 0;
    
    result = restic_list_snapshots(repo_id, &ids, &times, &hostnames, &count);
    if (result != RESTIC_OK) {
        print_error(result);
        restic_free_string(snapshot_id);
        restic_close(repo_id);
        return 1;
    }
    
    printf("Found %d snapshots:\n", count);
    for (int i = 0; i < count; i++) {
        printf("  ID: %s, Time: %s, Host: %s\n", 
               ids[i], times[i], hostnames[i]);
    }
    printf("\n");
    
    // Clean up snapshot arrays
    restic_free_snapshot_arrays(ids, times, hostnames, count);
    
    // Restore the backup
    printf("Restoring backup to /tmp/restore-test...\n");
    result = restic_restore(repo_id, snapshot_id, "/tmp/restore-test");
    if (result != RESTIC_OK) {
        print_error(result);
        restic_free_string(snapshot_id);
        restic_close(repo_id);
        return 1;
    }
    
    printf("Backup restored successfully\n\n");
    
    // Check repository integrity
    printf("Checking repository integrity...\n");
    int errors = 0;
    result = restic_check(repo_id, &errors);
    if (result != RESTIC_OK) {
        print_error(result);
    } else {
        printf("Repository check completed with %d errors\n\n", errors);
    }
    
    // Clean up
    restic_free_string(snapshot_id);
    restic_close(repo_id);
    
    // Clean up test files
    system("rm -rf /tmp/test-backup /tmp/restore-test /tmp/restic-test");
    
    printf("Example completed successfully!\n");
    return 0;
}