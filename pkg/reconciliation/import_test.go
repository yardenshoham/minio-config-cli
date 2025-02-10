package reconciliation

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	miniotestcontainer "github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestImport(t *testing.T) {
	t.Parallel()
	// create minio container
	ctx := context.Background()
	minioContainer, err := miniotestcontainer.Run(ctx, "minio/minio:RELEASE.2025-02-03T21-03-04Z")
	defer func() {
		err := testcontainers.TerminateContainer(minioContainer)
		require.NoError(t, err)
	}()
	require.NoError(t, err)

	endpoint, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	creds := credentials.NewStaticV4("minioadmin", "minioadmin", "")
	madminClient, err := madmin.NewWithOptions(endpoint, &madmin.Options{
		Secure: false,
		Creds:  creds,
	})
	require.NoError(t, err)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Secure: false,
		Creds:  creds,
	})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// actual test
	const readFoobarBucketPolicyName = "read-foobar-bucket"
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
	bucketsToImport := []bucket{
		{
			Name: "foobar",
			Lifecycle: map[string]any{
				"Rules": []map[string]any{
					{
						"ID":     "rule1",
						"Status": "Enabled",
						"Expiration": map[string]any{
							"Days": 1,
						},
					}},
			},
		},
	}
	ImportConfig := ImportConfig{
		Users:    usersToImport,
		Policies: policiesToImport,
		Buckets:  bucketsToImport,
	}

	users, err := madminClient.ListUsers(ctx)
	require.NoError(t, err)
	require.Empty(t, users)

	policies, err := madminClient.ListCannedPolicies(ctx)
	require.NoError(t, err)
	builtinPinnedPoliciesAmount := len(policies)

	buckets, err := minioClient.ListBuckets(ctx)
	require.NoError(t, err)
	require.Empty(t, buckets)

	// dry run should not change anything
	err = Import(ctx, logger, true, madminClient, minioClient, ImportConfig)
	require.NoError(t, err)

	users, err = madminClient.ListUsers(ctx)
	require.NoError(t, err)
	require.Empty(t, users)

	policies, err = madminClient.ListCannedPolicies(ctx)
	require.NoError(t, err)
	require.Len(t, policies, builtinPinnedPoliciesAmount)

	buckets, err = minioClient.ListBuckets(ctx)
	require.NoError(t, err)
	require.Empty(t, buckets)

	// twice to check idempotency
	for range 2 {
		err = Import(ctx, logger, false, madminClient, minioClient, ImportConfig)
		require.NoError(t, err)

		buckets, err = minioClient.ListBuckets(ctx)
		require.NoError(t, err)
		require.Len(t, buckets, len(bucketsToImport))
		require.Equal(t, bucketsToImport[0].Name, buckets[0].Name)

		lifecycle, err := minioClient.GetBucketLifecycle(ctx, bucketsToImport[0].Name)
		require.NoError(t, err)
		require.Len(t, lifecycle.Rules, 1)
		require.Equal(t, "rule1", lifecycle.Rules[0].ID)
		require.Equal(t, "Enabled", lifecycle.Rules[0].Status)
		require.Equal(t, 1, int(lifecycle.Rules[0].Expiration.Days))

		policies, err = madminClient.ListCannedPolicies(ctx)
		require.NoError(t, err)
		require.Len(t, policies, builtinPinnedPoliciesAmount+len(policiesToImport))
		require.Contains(t, policies, readFoobarBucketPolicyName)

		users, err = madminClient.ListUsers(ctx)
		require.NoError(t, err)
		require.Len(t, users, len(usersToImport))
		require.Contains(t, users, "first")
		require.Contains(t, users, "second")
		require.Equal(t, madmin.AccountEnabled, users["first"].Status)
		require.Equal(t, madmin.AccountDisabled, users["second"].Status)
		require.Equal(t, readFoobarBucketPolicyName, users["first"].PolicyName)
	}

	testdataConfigFile, err := os.Open("../../testdata/config.yaml")
	require.NoError(t, err)
	defer testdataConfigFile.Close()

	testdataConfig, err := LoadConfig(testdataConfigFile)
	require.NoError(t, err)
	err = Import(ctx, logger, false, madminClient, minioClient, *testdataConfig)
	require.NoError(t, err)
}
