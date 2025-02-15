package reconciliation

import (
	"context"
	"fmt"

	"github.com/minio/madmin-go/v3"
)

type user struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey,omitempty"`

	// Policies is an list of policy names to be applied for the user.
	Policies []string `yaml:"policies,omitempty"`

	// Status is either enabled or disabled, if not set it will be enabled.
	Status madmin.AccountStatus `yaml:"status"`
}

func (r *Reconciler) importUsers(ctx context.Context, users []user) error {
	r.logger.Info("importing users", "amount", len(users))
	for _, user := range users {
		setUserPayload := madmin.AddOrUpdateUserReq{
			SecretKey: user.SecretKey,
			Status:    user.Status,
		}
		// if the status is not set, default to enabled
		if setUserPayload.Status == "" {
			setUserPayload.Status = madmin.AccountEnabled
		}
		r.logger.Info("importing user", "accessKey", user.AccessKey, "status", setUserPayload.Status)

		if !r.dryRun {
			// we can't attach the policies in SetUserReq because of
			// https://github.com/minio/madmin-go/issues/216
			// so we will do it after the user is created
			err := r.madminClient.SetUserReq(ctx, user.AccessKey, setUserPayload)
			if err != nil {
				return fmt.Errorf("failed to set user %s: %w", user.AccessKey, err)
			}
			r.logger.Info("imported user", "accessKey", user.AccessKey)
		}
		if len(user.Policies) > 0 {
			err := r.attachUserPolicies(ctx, user)
			if err != nil {
				return fmt.Errorf("failed to attach policies to user %s: %w", user.AccessKey, err)
			}
		}
	}
	return nil
}

func (r *Reconciler) attachUserPolicies(ctx context.Context, user user) error {
	policyEntities, err := r.madminClient.GetPolicyEntities(ctx, madmin.PolicyEntitiesQuery{
		Users:  []string{user.AccessKey},
		Policy: user.Policies,
	})
	if err != nil {
		return fmt.Errorf("failed to get policy entities for user %s: %w", user.AccessKey, err)
	}
	policiesToAttachMap := make(map[string]struct{}, len(user.Policies))
	for _, policy := range user.Policies {
		policiesToAttachMap[policy] = struct{}{}
	}
	for _, policyUserMapping := range policyEntities.UserMappings {
		if policyUserMapping.User != user.AccessKey {
			panic("queried user's " + user.AccessKey + " policies but got user " + policyUserMapping.User)
		}
		for _, attachedPolicy := range policyUserMapping.Policies {
			r.logger.Info("user %s already has policy %s attached", user.AccessKey, attachedPolicy)
			delete(policiesToAttachMap, attachedPolicy)
		}
	}
	if len(policiesToAttachMap) == 0 {
		r.logger.Info("no policies left to attach", "accessKey", user.AccessKey, "policies", user.Policies)
		return nil
	}
	policiesToAttach := make([]string, 0, len(policiesToAttachMap))
	for policy := range policiesToAttachMap {
		policiesToAttach = append(policiesToAttach, policy)
	}
	r.logger.Info("attaching policies to user", "accessKey", user.AccessKey, "policies", policiesToAttach)
	if !r.dryRun {
		policyAssociationResp, err := r.madminClient.AttachPolicy(ctx, madmin.PolicyAssociationReq{
			Policies: policiesToAttach,
			User:     user.AccessKey,
		})
		if err != nil {
			return fmt.Errorf("failed to attach policies to user %s: %w", user.AccessKey, err)
		}
		r.logger.Info("attached policies to user", "accessKey", user.AccessKey, "policies", policyAssociationResp.PoliciesAttached)
	}
	return nil
}
