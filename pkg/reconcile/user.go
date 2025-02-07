package reconcile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/minio/madmin-go/v3"
)

type user struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey,omitempty"`

	// Policies is an list of policy names to be applied for the user.
	Policies []string             `yaml:"policies,omitempty"`
	Status   madmin.AccountStatus `yaml:"status"`
}

func importUsers(logger *slog.Logger, ctx context.Context, client *madmin.AdminClient, users []user) error {
	logger.Info("importing users", "amount", len(users))
	for _, user := range users {
		setUserPayload := madmin.AddOrUpdateUserReq{
			SecretKey: user.SecretKey,
			Status:    user.Status,
		}
		// if the status is not set, default to enabled
		if setUserPayload.Status == "" {
			setUserPayload.Status = madmin.AccountEnabled
		}
		logger.Info("importing user", "accessKey", user.AccessKey, "status", setUserPayload.Status)

		// we can't attach the policies in SetUserReq because of
		// https://github.com/minio/madmin-go/issues/216
		// so we will do it after the user is created
		err := client.SetUserReq(ctx, user.AccessKey, setUserPayload)
		if err != nil {
			return fmt.Errorf("failed to set user %s: %v", user.AccessKey, err)
		}
		logger.Info("imported user", "accessKey", user.AccessKey)
		if len(user.Policies) > 0 {
			logger.Info("attaching policies to user", "accessKey", user.AccessKey, "policies", user.Policies)
			policyAssociationResp, err := client.AttachPolicy(ctx, madmin.PolicyAssociationReq{
				Policies: user.Policies,
				User:     user.AccessKey,
			})
			if err != nil {
				return fmt.Errorf("failed to attach policies to user %s: %v", user.AccessKey, err)
			}
			logger.Info("attached policies to user", "accessKey", user.AccessKey, "policies", policyAssociationResp.PoliciesAttached)
		}
	}
	return nil
}
