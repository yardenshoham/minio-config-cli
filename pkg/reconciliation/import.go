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

func LoadConfig(r io.Reader) (*ImportConfig, error) {
	var b bytes.Buffer
	_, err := io.Copy(&b, r)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	err = validation.ValidateConfig(bytes.NewReader(b.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}
	config := &ImportConfig{}
	err = yaml.Unmarshal(b.Bytes(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}
	return config, nil
}

// Import imports the all resources from the config into the MinIO server. It is idempotent.
func Import(logger *slog.Logger, ctx context.Context, dryRun bool, madminClient *madmin.AdminClient, minioClient *minio.Client, config ImportConfig) error {
	err := importPolicies(logger, ctx, dryRun, madminClient, config.Policies)
	if err != nil {
		return fmt.Errorf("failed to import policies: %w", err)
	}
	err = importUsers(logger, ctx, dryRun, madminClient, config.Users)
	if err != nil {
		return fmt.Errorf("failed to import users: %w", err)
	}
	err = importBuckets(logger, ctx, dryRun, minioClient, config.Buckets)
	if err != nil {
		return fmt.Errorf("failed to import buckets: %w", err)
	}
	return nil
}
