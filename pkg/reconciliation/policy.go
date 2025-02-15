package reconciliation

import (
	"context"
	"fmt"
)

type policy struct {
	Name   string         `yaml:"name"`
	Policy map[string]any `yaml:"policy"`
}

func (r *Reconciler) importPolicies(ctx context.Context, policies []policy) error {
	r.logger.Info("importing policies", "amount", len(policies))
	for _, policy := range policies {
		r.logger.Info("importing policy", "name", policy.Name)
		asByteSlice, err := mapAnyToByteSlice(policy.Policy)
		if err != nil {
			return fmt.Errorf("failed to marshal policy %s: %w", policy.Name, err)
		}
		if !r.dryRun {
			err := r.madminClient.AddCannedPolicy(ctx, policy.Name, asByteSlice)
			if err != nil {
				return fmt.Errorf("failed to import policy %s: %w", policy.Name, err)
			}
			r.logger.Info("imported policy", "name", policy.Name)
		}
	}
	return nil
}
