package reconciliation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/madmin-go/v3"
)

type policy struct {
	Name   string `yaml:"name"`
	Policy string `yaml:"policy"`
}

func importPolicies(ctx context.Context, logger *slog.Logger, dryRun bool, client *madmin.AdminClient, policies []policy) error {
	logger.Info("importing policies", "amount", len(policies))
	for _, policy := range policies {
		logger.Info("importing policy", "name", policy.Name)
		if !dryRun {
			err := client.AddCannedPolicy(ctx, policy.Name, []byte(policy.Policy))
			if err != nil {
				return fmt.Errorf("failed to import policy %s: %w", policy.Name, err)
			}
			logger.Info("imported policy", "name", policy.Name)
		}
	}
	return nil
}
