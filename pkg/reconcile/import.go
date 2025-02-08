package reconcile

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"gopkg.in/yaml.v3"
)

type ImportConfig struct {
	Users    []user   `yaml:"users"`
	Policies []policy `yaml:"policies"`
	Buckets  []bucket `yaml:"buckets"`
}

func LoadConfig(r io.Reader) (*ImportConfig, error) {
	config := &ImportConfig{}
	err := yaml.NewDecoder(r).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %v", err)
	}
	return config, nil
}

func Import(logger *slog.Logger, ctx context.Context, madminClient *madmin.AdminClient, minioClient *minio.Client, config ImportConfig) error {
	err := importPolicies(logger, ctx, madminClient, config.Policies)
	if err != nil {
		return fmt.Errorf("failed to import policies: %v", err)
	}
	err = importUsers(logger, ctx, madminClient, config.Users)
	if err != nil {
		return fmt.Errorf("failed to import users: %v", err)
	}
	err = importBuckets(logger, ctx, minioClient, config.Buckets)
	if err != nil {
		return fmt.Errorf("failed to import buckets: %v", err)
	}
	return nil
}
