#ifndef RESTICLIB_HPP
#define RESTICLIB_HPP

#include <string>
#include <vector>
#include <memory>
#include <stdexcept>
#include <cstdlib>
#include "resticlib.h"

namespace ResticLib {

// Exception class for ResticLib errors
class ResticException : public std::runtime_error {
public:
    ResticException(int error_code, const std::string& message)
        : std::runtime_error(message), error_code_(error_code) {}
    
    int error_code() const { return error_code_; }
    
private:
    int error_code_;
};

// Configuration structure for repository
struct Config {
    std::string repoUrl;
    std::string backend;  
    std::string password;
    std::string accessKey;  // optional
    std::string secretKey;  // optional
    int parallelism = 4;
};

// Snapshot information
struct Snapshot {
    std::string id;
    std::string time;
    std::string hostname;
};

// RAII wrapper for C string
class CString {
public:
    explicit CString(char* ptr) : ptr_(ptr) {}
    ~CString() { if (ptr_) restic_free_string(ptr_); }
    
    CString(const CString&) = delete;
    CString& operator=(const CString&) = delete;
    
    CString(CString&& other) noexcept : ptr_(other.ptr_) { other.ptr_ = nullptr; }
    CString& operator=(CString&& other) noexcept {
        if (this != &other) {
            if (ptr_) restic_free_string(ptr_);
            ptr_ = other.ptr_;
            other.ptr_ = nullptr;
        }
        return *this;
    }
    
    const char* get() const { return ptr_; }
    std::string str() const { return ptr_ ? std::string(ptr_) : std::string(); }
    
private:
    char* ptr_;
};

// Main Repository class
class Repository {
public:
    // Constructor for initializing a new repository
    explicit Repository(const Config& config, bool init_repo = false) {
        const char* access_key = config.accessKey.empty() ? nullptr : config.accessKey.c_str();
        const char* secret_key = config.secretKey.empty() ? nullptr : config.secretKey.c_str();
        
        int result;
        if (init_repo) {
            result = restic_init(
                const_cast<char*>(config.repoUrl.c_str()),
                const_cast<char*>(config.backend.c_str()),
                const_cast<char*>(config.password.c_str()),
                const_cast<char*>(access_key),
                const_cast<char*>(secret_key),
                config.parallelism
            );
        } else {
            result = restic_open(
                const_cast<char*>(config.repoUrl.c_str()),
                const_cast<char*>(config.backend.c_str()),
                const_cast<char*>(config.password.c_str()),
                const_cast<char*>(access_key),
                const_cast<char*>(secret_key),
                config.parallelism
            );
        }
        
        if (result < 0) {
            CString error_msg(restic_get_error_message(result));
            throw ResticException(result, error_msg.str());
        }
        
        repo_id_ = result;
    }
    
    // Destructor
    ~Repository() {
        if (repo_id_ >= 0) {
            restic_close(repo_id_);
        }
    }
    
    // Disable copy constructor and assignment
    Repository(const Repository&) = delete;
    Repository& operator=(const Repository&) = delete;
    
    // Enable move constructor and assignment
    Repository(Repository&& other) noexcept : repo_id_(other.repo_id_) {
        other.repo_id_ = -1;
    }
    
    Repository& operator=(Repository&& other) noexcept {
        if (this != &other) {
            if (repo_id_ >= 0) {
                restic_close(repo_id_);
            }
            repo_id_ = other.repo_id_;
            other.repo_id_ = -1;
        }
        return *this;
    }
    
    // Create a backup
    std::string backup(const std::vector<std::string>& paths, const std::vector<std::string>& tags = {}) {
        if (paths.empty()) {
            throw ResticException(RESTIC_ERROR_INVALID_PARAMS, "Paths cannot be empty");
        }
        
        // Convert paths to C array
        std::vector<char*> c_paths;
        c_paths.reserve(paths.size());
        for (const auto& path : paths) {
            c_paths.push_back(const_cast<char*>(path.c_str()));
        }
        
        // Convert tags to C array
        std::vector<char*> c_tags;
        c_tags.reserve(tags.size());
        for (const auto& tag : tags) {
            c_tags.push_back(const_cast<char*>(tag.c_str()));
        }
        
        char* snapshot_id = nullptr;
        int result = restic_backup(
            repo_id_,
            c_paths.data(),
            static_cast<int>(c_paths.size()),
            tags.empty() ? nullptr : c_tags.data(),
            static_cast<int>(c_tags.size()),
            &snapshot_id
        );
        
        if (result != RESTIC_OK) {
            CString error_msg(restic_get_error_message(result));
            throw ResticException(result, error_msg.str());
        }
        
        CString snapshot_id_wrapper(snapshot_id);
        return snapshot_id_wrapper.str();
    }
    
    // Restore a snapshot
    void restore(const std::string& snapshot_id, const std::string& target_dir) {
        int result = restic_restore(
            repo_id_,
            const_cast<char*>(snapshot_id.c_str()),
            const_cast<char*>(target_dir.c_str())
        );
        
        if (result != RESTIC_OK) {
            CString error_msg(restic_get_error_message(result));
            throw ResticException(result, error_msg.str());
        }
    }
    
    // List all snapshots
    std::vector<Snapshot> listSnapshots() {
        char** ids = nullptr;
        char** times = nullptr;
        char** hostnames = nullptr;
        int count = 0;
        
        int result = restic_list_snapshots(repo_id_, &ids, &times, &hostnames, &count);
        
        if (result != RESTIC_OK) {
            CString error_msg(restic_get_error_message(result));
            throw ResticException(result, error_msg.str());
        }
        
        std::vector<Snapshot> snapshots;
        snapshots.reserve(count);
        
        for (int i = 0; i < count; i++) {
            Snapshot snapshot;
            snapshot.id = ids[i] ? std::string(ids[i]) : "";
            snapshot.time = times[i] ? std::string(times[i]) : "";
            snapshot.hostname = hostnames[i] ? std::string(hostnames[i]) : "";
            snapshots.push_back(std::move(snapshot));
        }
        
        // Free the arrays
        restic_free_snapshot_arrays(ids, times, hostnames, count);
        
        return snapshots;
    }
    
    // Check repository integrity
    int check() {
        int errors = 0;
        int result = restic_check(repo_id_, &errors);
        
        if (result != RESTIC_OK) {
            CString error_msg(restic_get_error_message(result));
            throw ResticException(result, error_msg.str());
        }
        
        return errors;
    }
    
    // Get library version
    static std::string getVersion() {
        CString version(restic_get_version());
        return version.str();
    }
    
private:
    int repo_id_ = -1;
};

} // namespace ResticLib

#endif // RESTICLIB_HPP