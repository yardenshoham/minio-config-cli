package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

type bucket struct {
	Name      string         `yaml:"name"`
	Lifecycle map[string]any `yaml:"lifecycle,omitempty"`
	Policy    map[string]any `yaml:"policy,omitempty"`
}

func (r *Reconciler) importBuckets(ctx context.Context, buckets []bucket) error {
	r.logger.Info("importing buckets", "amount", len(buckets))
	for _, bucket := range buckets {
		exists, err := r.minioClient.BucketExists(ctx, bucket.Name)
		if err != nil {
			return fmt.Errorf("failed to check if bucket %s exists: %w", bucket.Name, err)
		}
		if exists {
			r.logger.Info("bucket already exists", "name", bucket.Name)
		} else {
			r.logger.Info("importing bucket", "name", bucket.Name)
			if !r.dryRun {
				err = r.minioClient.MakeBucket(ctx, bucket.Name, minio.MakeBucketOptions{})
				if err != nil {
					return fmt.Errorf("failed to import bucket %s: %w", bucket.Name, err)
				}
				r.logger.Info("imported bucket", "name", bucket.Name)
			}
		}
		if len(bucket.Lifecycle) > 0 {
			r.logger.Info("importing bucket lifecycle", "name", bucket.Name)
			lifecycleConfiguration := lifecycle.NewConfiguration()
			asJSON, err := json.Marshal(bucket.Lifecycle)
			if err != nil {
				return fmt.Errorf("failed to marshal lifecycle configuration for bucket %s: %w", bucket.Name, err)
			}
			err = json.Unmarshal(asJSON, &lifecycleConfiguration)
			if err != nil {
				return fmt.Errorf("failed to unmarshal lifecycle configuration %s for bucket %s: %w", bucket.Lifecycle, bucket.Name, err)
			}
			if !r.dryRun {
				err = r.minioClient.SetBucketLifecycle(ctx, bucket.Name, lifecycleConfiguration)
				if err != nil {
					return fmt.Errorf("failed to set lifecycle configuration %s for bucket %s: %w", bucket.Lifecycle, bucket.Name, err)
				}
				r.logger.Info("imported bucket lifecycle", "name", bucket.Name)
			}
		}
		if len(bucket.Policy) > 0 {
			r.logger.Info("importing bucket policy", "name", bucket.Name)
			asByteSlice, err := mapAnyToByteSlice(bucket.Policy)
			if err != nil {
				return fmt.Errorf("failed to marshal policy for bucket %s: %w", bucket.Name, err)
			}
			if !r.dryRun {
				err = r.minioClient.SetBucketPolicy(ctx, bucket.Name, string(asByteSlice))
				if err != nil {
					return fmt.Errorf("failed to set policy for bucket %s: %w", bucket.Name, err)
				}
				r.logger.Info("imported bucket policy", "name", bucket.Name)
			}
		}
	}
	return nil
}
