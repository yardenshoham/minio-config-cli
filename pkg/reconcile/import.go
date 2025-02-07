package reconcile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/madmin-go/v3"
)

type ImportConfig struct {
	Users    []user   `yaml:"users"`
	Policies []policy `yaml:"policies"`
}

func Import(logger *slog.Logger, ctx context.Context, client *madmin.AdminClient, config ImportConfig) error {
	err := importPolicies(logger, ctx, client, config.Policies)
	if err != nil {
		return fmt.Errorf("failed to import policies: %v", err)
	}
	err = importUsers(logger, ctx, client, config.Users)
	if err != nil {
		return fmt.Errorf("failed to import users: %v", err)
	}
	return nil
}
