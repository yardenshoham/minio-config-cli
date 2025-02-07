package reconcile

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

func importPolicies(logger *slog.Logger, ctx context.Context, client *madmin.AdminClient, policies []policy) error {
	logger.Info("importing policies", "amount", len(policies))
	for _, policy := range policies {
		logger.Info("importing policy", "name", policy.Name)
		err := client.AddCannedPolicy(ctx, policy.Name, []byte(policy.Policy))
		if err != nil {
			return fmt.Errorf("failed to import policy %s: %v", policy.Name, err)
		}
		logger.Info("imported policy", "name", policy.Name)
	}
	return nil
}
