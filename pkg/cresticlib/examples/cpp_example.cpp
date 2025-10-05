#include <iostream>
#include <vector>
#include <string>
#include <cstdlib>
#include "../pkg/cresticlib/ResticLib.hpp"

int main() {
    std::cout << "ResticLib C++ Example\n";
    std::cout << "====================\n\n";
    
    try {
        // Get library version
        std::cout << "Library version: " << ResticLib::Repository::getVersion() << "\n\n";
        
        // Create configuration
        ResticLib::Config config;
        config.repoUrl = "/tmp/restic-test-cpp";
        config.backend = "local";
        config.password = "testpassword123";
        config.parallelism = 2;
        
        // Initialize a new repository
        std::cout << "Initializing repository...\n";
        ResticLib::Repository repo(config, true);  // true = initialize new repo
        std::cout << "Repository initialized successfully\n\n";
        
        // Create some test directories and files
        std::system("mkdir -p /tmp/test-backup-cpp/documents /tmp/test-backup-cpp/images");
        std::system("echo 'C++ Example File' > /tmp/test-backup-cpp/readme.txt");
        std::system("echo 'Document content' > /tmp/test-backup-cpp/documents/doc1.txt");
        std::system("echo 'Image data' > /tmp/test-backup-cpp/images/image1.jpg");
        
        // Create a backup
        std::cout << "Creating backup...\n";
        std::vector<std::string> paths = {"/tmp/test-backup-cpp"};
        std::vector<std::string> tags = {"cpp-example", "automated", "test"};
        
        std::string snapshot_id = repo.backup(paths, tags);
        std::cout << "Backup created with snapshot ID: " << snapshot_id << "\n\n";
        
        // List snapshots
        std::cout << "Listing snapshots...\n";
        auto snapshots = repo.listSnapshots();
        
        std::cout << "Found " << snapshots.size() << " snapshots:\n";
        for (const auto& snapshot : snapshots) {
            std::cout << "  ID: " << snapshot.id 
                      << ", Time: " << snapshot.time 
                      << ", Host: " << snapshot.hostname << "\n";
        }
        std::cout << "\n";
        
        // Restore the backup
        std::cout << "Restoring backup to /tmp/restore-test-cpp...\n";
        repo.restore(snapshot_id, "/tmp/restore-test-cpp");
        std::cout << "Backup restored successfully\n\n";
        
        // Check repository integrity
        std::cout << "Checking repository integrity...\n";
        int errors = repo.check();
        std::cout << "Repository check completed with " << errors << " errors\n\n";
        
        // Demonstrate backup with different paths and tags
        std::cout << "Creating second backup with multiple paths...\n";
        std::vector<std::string> multi_paths = {
            "/tmp/test-backup-cpp/documents", 
            "/tmp/test-backup-cpp/readme.txt"
        };
        std::vector<std::string> new_tags = {"partial-backup", "documents-only"};
        
        std::string second_snapshot = repo.backup(multi_paths, new_tags);
        std::cout << "Second backup created: " << second_snapshot << "\n\n";
        
        // List snapshots again
        std::cout << "Updated snapshot list:\n";
        snapshots = repo.listSnapshots();
        for (const auto& snapshot : snapshots) {
            std::cout << "  ID: " << snapshot.id << ", Host: " << snapshot.hostname << "\n";
        }
        std::cout << "\n";
        
        std::cout << "Example completed successfully!\n";
        
    } catch (const ResticLib::ResticException& e) {
        std::cerr << "ResticLib Error [" << e.error_code() << "]: " << e.what() << "\n";
        return 1;
    } catch (const std::exception& e) {
        std::cerr << "Standard Error: " << e.what() << "\n";
        return 1;
    }
    
    // Clean up test files
    std::system("rm -rf /tmp/test-backup-cpp /tmp/restore-test-cpp /tmp/restic-test-cpp");
    
    return 0;
}