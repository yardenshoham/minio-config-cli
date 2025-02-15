package reconciliation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/yardenshoham/minio-config-cli/pkg/validation"
	"gopkg.in/yaml.v3"
)

type ImportConfig struct {
	Users    []user   `yaml:"users"`
	Policies []policy `yaml:"policies"`
	Buckets  []bucket `yaml:"buckets"`
}

// LoadConfig loads the config file into a struct.
func LoadConfig(r io.Reader) (*ImportConfig, error) {
	var readerText bytes.Buffer
	_, err := io.Copy(&readerText, r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	err = validation.ValidateConfig(bytes.NewReader(readerText.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}
	config := &ImportConfig{}
	err = yaml.Unmarshal(readerText.Bytes(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	return config, nil
}

// Reconciler handles the configuration of a reconciliation run.
type Reconciler struct {
	logger       *slog.Logger
	madminClient *madmin.AdminClient
	minioClient  *minio.Client
	dryRun       bool
}

// NewReconciler creates a new Reconciler instance.
func NewReconciler(logger *slog.Logger, madminClient *madmin.AdminClient, minioClient *minio.Client, dryRun bool) *Reconciler {
	return &Reconciler{
		logger:       logger,
		madminClient: madminClient,
		minioClient:  minioClient,
		dryRun:       dryRun,
	}
}

// Import imports the all resources from the config into the MinIO server. It is idempotent.
func (r *Reconciler) Import(ctx context.Context, config ImportConfig) error {
	err := r.importPolicies(ctx, config.Policies)
	if err != nil {
		return fmt.Errorf("failed to import policies: %w", err)
	}
	err = r.importUsers(ctx, config.Users)
	if err != nil {
		return fmt.Errorf("failed to import users: %w", err)
	}
	err = r.importBuckets(ctx, config.Buckets)
	if err != nil {
		return fmt.Errorf("failed to import buckets: %w", err)
	}
	return nil
}
