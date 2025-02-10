package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

type bucket struct {
	Name      string         `yaml:"name"`
	Lifecycle map[string]any `yaml:"lifecycle,omitempty"`
}

func importBuckets(ctx context.Context, logger *slog.Logger, dryRun bool, client *minio.Client, buckets []bucket) error {
	logger.Info("importing buckets", "amount", len(buckets))
	for _, bucket := range buckets {
		exists, err := client.BucketExists(ctx, bucket.Name)
		if err != nil {
			return fmt.Errorf("failed to check if bucket %s exists: %w", bucket.Name, err)
		}
		if exists {
			logger.Info("bucket already exists", "name", bucket.Name)
		} else {
			logger.Info("importing bucket", "name", bucket.Name)
			if !dryRun {
				err = client.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{})
				if err != nil {
					return fmt.Errorf("failed to import bucket %s: %w", bucket.Name, err)
				}
				logger.Info("imported bucket", "name", bucket.Name)
			}
		}
		if len(bucket.Lifecycle) > 0 {
			logger.Info("importing bucket lifecycle", "name", bucket.Name)
			lifecycleConfiguration := lifecycle.NewConfiguration()
			asJSON, err := json.Marshal(bucket.Lifecycle)
			if err != nil {
				return fmt.Errorf("failed to marshal lifecycle configuration for bucket %s: %w", bucket.Name, err)
			}
			err = json.Unmarshal(asJSON, &lifecycleConfiguration)
			if err != nil {
				return fmt.Errorf("failed to unmarshal lifecycle configuration %s for bucket %s: %w", bucket.Lifecycle, bucket.Name, err)
			}
			if !dryRun {
				err = client.SetBucketLifecycle(ctx, bucket.Name, lifecycleConfiguration)
				if err != nil {
					return fmt.Errorf("failed to set lifecycle configuration %s for bucket %s: %w", bucket.Lifecycle, bucket.Name, err)
				}
				logger.Info("imported bucket lifecycle", "name", bucket.Name)
			}
		}
	}
	return nil
}
