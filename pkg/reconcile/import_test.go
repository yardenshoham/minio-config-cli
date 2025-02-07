package reconcile

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestImport(t *testing.T) {
	// create minio container
	ctx := context.Background()
	minioContainer, err := minio.Run(ctx, "minio/minio:RELEASE.2025-02-03T21-03-04Z")
	defer func() {
		err := testcontainers.TerminateContainer(minioContainer)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	url, err := minioContainer.ConnectionString(ctx)
	assert.NoError(t, err)

	client, err := madmin.NewWithOptions(url, &madmin.Options{
		Secure: false,
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
	})
	assert.NoError(t, err)

	const readFoobarBucketPolicyName = "read-foobar-bucket"

	// actual test
	policiesToImport := []policy{
		{
			Name: readFoobarBucketPolicyName,
			Policy: `{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Action": [
							"s3:GetObject"
						],
						"Resource": [
							"arn:aws:s3:::foobar/*"
						]
					}
				]
			}`,
		},
	}
	usersToImport := []user{
		{
			AccessKey: "first",
			SecretKey: "secretnicewowwow",
			Policies:  []string{readFoobarBucketPolicyName},
		},
		{
			AccessKey: "second",
			SecretKey: "secretnicewowwow",
			Status:    madmin.AccountDisabled,
		},
	}
	ImportConfig := ImportConfig{
		Users:    usersToImport,
		Policies: policiesToImport,
	}
	users, err := client.ListUsers(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	policies, err := client.ListCannedPolicies(ctx)
	assert.NoError(t, err)
	builtinPinnedPoliciesAmount := len(policies)

	err = Import(slog.New(slog.NewTextHandler(os.Stdout, nil)), ctx, client, ImportConfig)
	assert.NoError(t, err)

	policies, err = client.ListCannedPolicies(ctx)
	assert.NoError(t, err)
	assert.Len(t, policies, builtinPinnedPoliciesAmount+len(policiesToImport))
	assert.Contains(t, policies, readFoobarBucketPolicyName)

	users, err = client.ListUsers(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, len(usersToImport))
	assert.Contains(t, users, "first")
	assert.Contains(t, users, "second")
	assert.Equal(t, madmin.AccountEnabled, users["first"].Status)
	assert.Equal(t, madmin.AccountDisabled, users["second"].Status)
	assert.Equal(t, readFoobarBucketPolicyName, users["first"].PolicyName)
}
