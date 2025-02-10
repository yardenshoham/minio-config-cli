package reconciliation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/minio/madmin-go/v3"
)

type policy struct {
	Name   string         `yaml:"name"`
	Policy map[string]any `yaml:"policy"`
}

func importPolicies(ctx context.Context, logger *slog.Logger, dryRun bool, client *madmin.AdminClient, policies []policy) error {
	logger.Info("importing policies", "amount", len(policies))
	for _, policy := range policies {
		logger.Info("importing policy", "name", policy.Name)
		asJSON, err := json.Marshal(policy.Policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy %s: %w", policy.Name, err)
		}
		if !dryRun {
			err := client.AddCannedPolicy(ctx, policy.Name, []byte(asJSON))
			if err != nil {
				return fmt.Errorf("failed to import policy %s: %w", policy.Name, err)
			}
			logger.Info("imported policy", "name", policy.Name)
		}
	}
	return nil
}
