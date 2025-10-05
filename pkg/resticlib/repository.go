package resticlib

import (
	"context"
	"fmt"

	"github.com/restic/restic/internal/backend"
	"github.com/restic/restic/internal/backend/azure"
	"github.com/restic/restic/internal/backend/b2"
	"github.com/restic/restic/internal/backend/gs"
	"github.com/restic/restic/internal/backend/local"
	"github.com/restic/restic/internal/backend/location"
	"github.com/restic/restic/internal/backend/rclone"
	"github.com/restic/restic/internal/backend/rest"
	"github.com/restic/restic/internal/backend/s3"
	"github.com/restic/restic/internal/backend/sftp"
	"github.com/restic/restic/internal/backend/swift"
	"github.com/restic/restic/internal/errors"
	"github.com/restic/restic/internal/repository"
	"github.com/restic/restic/internal/restic"
)

// repositoryImpl implements the Repository interface
type repositoryImpl struct {
	repo   *repository.Repository
	cfg    Config
	logger Logger
}

// getBackendRegistry creates and returns a backend registry with all supported backends
func getBackendRegistry() *location.Registry {
	registry := location.NewRegistry()
	registry.Register(azure.NewFactory())
	registry.Register(b2.NewFactory())
	registry.Register(gs.NewFactory())
	registry.Register(local.NewFactory())
	registry.Register(rclone.NewFactory())
	registry.Register(rest.NewFactory())
	registry.Register(s3.NewFactory())
	registry.Register(sftp.NewFactory())
	registry.Register(swift.NewFactory())
	return registry
}

// createBackend creates a backend based on the configuration
func createBackend(ctx context.Context, cfg Config) (backend.Backend, error) {
	registry := getBackendRegistry()
	loc, err := location.Parse(registry, cfg.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Extract credentials from config if available
	var options map[string]string
	if cfg.Credentials != nil {
		options = make(map[string]string)
		if cfg.Credentials.AccessKey != "" {
			options["access-key-id"] = cfg.Credentials.AccessKey
			options["secret-access-key"] = cfg.Credentials.SecretKey
		}
		if cfg.Credentials.Token != "" {
			options["session-token"] = cfg.Credentials.Token
		}
	}

	// Logger function for backend (can be nil)
	var loggerFunc func(string, ...interface{})
	
	// Create backend based on scheme
	switch loc.Scheme {
	case "local":
		return local.Create(ctx, loc.Config.(local.Config), loggerFunc)
	case "s3":
		return s3.Create(ctx, loc.Config.(s3.Config), nil, loggerFunc)
	case "azure":
		return azure.Create(ctx, loc.Config.(azure.Config), nil, loggerFunc)
	case "gs":
		return gs.Create(ctx, loc.Config.(gs.Config), nil, loggerFunc)
	case "b2":
		return b2.Create(ctx, loc.Config.(b2.Config), nil, loggerFunc)
	case "sftp":
		return sftp.Create(ctx, loc.Config.(sftp.Config), loggerFunc)
	case "swift":
		return swift.Open(ctx, loc.Config.(swift.Config), nil, loggerFunc)
	case "rest":
		return rest.Create(ctx, loc.Config.(rest.Config), nil, loggerFunc)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", loc.Scheme)
	}
}

// openBackend opens an existing backend
func openBackend(ctx context.Context, cfg Config) (backend.Backend, error) {
	registry := getBackendRegistry()
	loc, err := location.Parse(registry, cfg.RepoURL)
	if err != nil {
		return nil, fmt.Errorf("invalid repository URL: %w", err)
	}

	// Extract credentials from config if available
	var options map[string]string
	if cfg.Credentials != nil {
		options = make(map[string]string)
		if cfg.Credentials.AccessKey != "" {
			options["access-key-id"] = cfg.Credentials.AccessKey
			options["secret-access-key"] = cfg.Credentials.SecretKey
		}
		if cfg.Credentials.Token != "" {
			options["session-token"] = cfg.Credentials.Token
		}
	}

	// Logger function for backend (can be nil)
	var loggerFunc func(string, ...interface{})
	
	// Open backend based on scheme
	switch loc.Scheme {
	case "local":
		return local.Open(ctx, loc.Config.(local.Config), loggerFunc)
	case "s3":
		return s3.Open(ctx, loc.Config.(s3.Config), nil, loggerFunc)
	case "azure":
		return azure.Open(ctx, loc.Config.(azure.Config), nil, loggerFunc)
	case "gs":
		return gs.Open(ctx, loc.Config.(gs.Config), nil, loggerFunc)
	case "b2":
		return b2.Open(ctx, loc.Config.(b2.Config), nil, loggerFunc)
	case "sftp":
		return sftp.Open(ctx, loc.Config.(sftp.Config), loggerFunc)
	case "swift":
		return swift.Open(ctx, loc.Config.(swift.Config), nil, loggerFunc)
	case "rest":
		return rest.Open(ctx, loc.Config.(rest.Config), nil, loggerFunc)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", loc.Scheme)
	}
}

// Init initializes a new repository with the given configuration
func Init(ctx context.Context, cfg Config) (Repository, error) {
	if cfg.Password == nil || len(cfg.Password) == 0 {
		return nil, errors.New("password is required")
	}

	// Create backend
	be, err := createBackend(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend: %w", err)
	}

	// Create repository wrapper
	repo, err := repository.New(be, repository.Options{})
	if err != nil {
		_ = be.Close()
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Initialize repository with password
	version := uint(restic.MaxRepoVersion)
	err = repo.Init(ctx, version, string(cfg.Password), nil)
	if err != nil {
		_ = be.Close()
		return nil, fmt.Errorf("failed to initialize repository: %w", err)
	}

	return &repositoryImpl{
		repo:   repo,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// Open opens an existing repository with the given configuration
func Open(ctx context.Context, cfg Config) (Repository, error) {
	if cfg.Password == nil || len(cfg.Password) == 0 {
		return nil, errors.New("password is required")
	}

	// Open backend
	be, err := openBackend(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to open backend: %w", err)
	}

	// Create repository wrapper
	repo, err := repository.New(be, repository.Options{})
	if err != nil {
		_ = be.Close()
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Search for key and decrypt with password
	err = repo.SearchKey(ctx, string(cfg.Password), 0, "")
	if err != nil {
		_ = be.Close()
		return nil, fmt.Errorf("failed to open repository (invalid password?): %w", err)
	}

	return &repositoryImpl{
		repo:   repo,
		cfg:    cfg,
		logger: cfg.Logger,
	}, nil
}

// Close closes the repository connection
func (r *repositoryImpl) Close() error {
	return r.repo.Close()
}

// Additional helper methods will be implemented in subsequent files...

// logf logs a message if a logger is available
func (r *repositoryImpl) logf(level string, msg string, args ...interface{}) {
	if r.logger == nil {
		return
	}
	
	switch level {
	case "debug":
		r.logger.Debug(msg, args...)
	case "info":
		r.logger.Info(msg, args...)
	case "warn":
		r.logger.Warn(msg, args...)
	case "error":
		r.logger.Error(msg, args...)
	}
}