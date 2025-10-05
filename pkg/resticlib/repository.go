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
		if cfg, ok := loc.Config.(*local.Config); ok {
			return local.Create(ctx, *cfg, loggerFunc)
		} else if cfg, ok := loc.Config.(local.Config); ok {
			return local.Create(ctx, cfg, loggerFunc)
		}
		return nil, fmt.Errorf("invalid local config type")
	case "s3":
		if cfg, ok := loc.Config.(*s3.Config); ok {
			return s3.Create(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(s3.Config); ok {
			return s3.Create(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid s3 config type")
	case "azure":
		if cfg, ok := loc.Config.(*azure.Config); ok {
			return azure.Create(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(azure.Config); ok {
			return azure.Create(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid azure config type")
	case "gs":
		if cfg, ok := loc.Config.(*gs.Config); ok {
			return gs.Create(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(gs.Config); ok {
			return gs.Create(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid gs config type")
	case "b2":
		if cfg, ok := loc.Config.(*b2.Config); ok {
			return b2.Create(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(b2.Config); ok {
			return b2.Create(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid b2 config type")
	case "sftp":
		if cfg, ok := loc.Config.(*sftp.Config); ok {
			return sftp.Create(ctx, *cfg, loggerFunc)
		} else if cfg, ok := loc.Config.(sftp.Config); ok {
			return sftp.Create(ctx, cfg, loggerFunc)
		}
		return nil, fmt.Errorf("invalid sftp config type")
	case "swift":
		if cfg, ok := loc.Config.(*swift.Config); ok {
			return swift.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(swift.Config); ok {
			return swift.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid swift config type")
	case "rest":
		if cfg, ok := loc.Config.(*rest.Config); ok {
			return rest.Create(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(rest.Config); ok {
			return rest.Create(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid rest config type")
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
		if cfg, ok := loc.Config.(*local.Config); ok {
			return local.Open(ctx, *cfg, loggerFunc)
		} else if cfg, ok := loc.Config.(local.Config); ok {
			return local.Open(ctx, cfg, loggerFunc)
		}
		return nil, fmt.Errorf("invalid local config type")
	case "s3":
		if cfg, ok := loc.Config.(*s3.Config); ok {
			return s3.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(s3.Config); ok {
			return s3.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid s3 config type")
	case "azure":
		if cfg, ok := loc.Config.(*azure.Config); ok {
			return azure.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(azure.Config); ok {
			return azure.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid azure config type")
	case "gs":
		if cfg, ok := loc.Config.(*gs.Config); ok {
			return gs.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(gs.Config); ok {
			return gs.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid gs config type")
	case "b2":
		if cfg, ok := loc.Config.(*b2.Config); ok {
			return b2.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(b2.Config); ok {
			return b2.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid b2 config type")
	case "sftp":
		if cfg, ok := loc.Config.(*sftp.Config); ok {
			return sftp.Open(ctx, *cfg, loggerFunc)
		} else if cfg, ok := loc.Config.(sftp.Config); ok {
			return sftp.Open(ctx, cfg, loggerFunc)
		}
		return nil, fmt.Errorf("invalid sftp config type")
	case "swift":
		if cfg, ok := loc.Config.(*swift.Config); ok {
			return swift.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(swift.Config); ok {
			return swift.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid swift config type")
	case "rest":
		if cfg, ok := loc.Config.(*rest.Config); ok {
			return rest.Open(ctx, *cfg, nil, loggerFunc)
		} else if cfg, ok := loc.Config.(rest.Config); ok {
			return rest.Open(ctx, cfg, nil, loggerFunc)
		}
		return nil, fmt.Errorf("invalid rest config type")
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