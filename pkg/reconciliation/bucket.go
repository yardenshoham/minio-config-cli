package reconciliation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

type bucket struct {
	Name string `yaml:"name"`
}

func importBuckets(logger *slog.Logger, ctx context.Context, dryRun bool, client *minio.Client, buckets []bucket) error {
	logger.Info("importing buckets", "amount", len(buckets))
	for _, bucket := range buckets {
		exists, err := client.BucketExists(ctx, bucket.Name)
		if err != nil {
			return fmt.Errorf("failed to check if bucket %s exists: %w", bucket.Name, err)
		}
		if exists {
			logger.Info("bucket already exists", "name", bucket.Name)
			continue
		}
		logger.Info("importing bucket", "name", bucket.Name)
		if !dryRun {
			err = client.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("failed to import bucket %s: %w", bucket.Name, err)
			}
			logger.Info("imported bucket", "name", bucket.Name)
		}
	}
	return nil
}
