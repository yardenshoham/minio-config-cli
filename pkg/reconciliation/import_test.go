package reconciliation

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	miniotestcontainer "github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/yardenshoham/minio-config-cli/pkg/validation"
)

func testSetup(t *testing.T, customizers ...testcontainers.ContainerCustomizer) (context.Context, *madmin.AdminClient, *minio.Client, *slog.Logger, *miniotestcontainer.MinioContainer) {
	t.Helper()
	ctx := t.Context()
	// don't wait for minio to be up, we want to test our waiting code
	noWaitStrategy := wait.ForNop(func(_ context.Context, _ wait.StrategyTarget) error { return nil })
	finalCustomizers := append(customizers, testcontainers.WithWaitStrategy(noWaitStrategy))
	minioContainer, err := miniotestcontainer.Run(ctx, "coollabsio/minio:RELEASE.2025-10-15T17-29-55Z", finalCustomizers...)
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
	return ctx, madminClient, minioClient, logger, minioContainer
}

func TestImportWhenNotReady(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip()
	}
	ctx, madminClient, minioClient, logger, minioContainer := testSetup(t, testcontainers.WithEntrypoint("sh"), testcontainers.WithCmd("-c", "sleep 110 && minio server /data"))
	defer func() {
		err := testcontainers.TerminateContainer(minioContainer)
		require.NoError(t, err)
	}()
	importConfig := ImportConfig{
		Policies: []policy{
			{
				Name: "read-foobar-bucket",
				Policy: map[string]any{
					"Version": "2012-10-17",
					"Statement": []map[string]any{
						{
							"Effect": "Allow",
							"Action": []string{
								"s3:GetObject",
							},
							"Resource": "arn:aws:s3:::foobar/*",
						},
					},
				},
			},
		},
	}

	err := Import(ctx, logger, false, madminClient, minioClient, importConfig)
	require.NoError(t, err)
}

func TestImport(t *testing.T) {
	t.Parallel()
	ctx, madminClient, minioClient, logger, minioContainer := testSetup(t)
	defer func() {
		err := testcontainers.TerminateContainer(minioContainer)
		require.NoError(t, err)
	}()
	const readFoobarBucketPolicyName = "read-foobar-bucket"
	policiesToImport := []policy{
		{
			Name: readFoobarBucketPolicyName,
			Policy: map[string]any{
				"Version": "2012-10-17",
				"Statement": []map[string]any{
					{
						"Effect": "Allow",
						"Action": []string{
							"s3:GetObject",
						},
						"Resource": "arn:aws:s3:::foobar/*",
					},
				},
			},
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
					},
				},
			},
			Quota: map[string]any{
				"Size": 10737418240,
			},
			Policy: map[string]any{
				"Version": "2012-10-17",
				"Statement": []map[string]any{
					{
						"Effect": "Allow",
						"Action": []string{
							"s3:GetObject",
							"s3:ListBucket",
						},
						"Resource": []string{"arn:aws:s3:::*"},
						"Principal": map[string]any{
							"AWS": []string{"*"},
						},
					},
				},
			},
		},
	}
	importConfig := ImportConfig{
		Users:    usersToImport,
		Policies: policiesToImport,
		Buckets:  bucketsToImport,
	}

	asJSON, err := json.Marshal(importConfig)
	require.NoError(t, err)
	err = validation.ValidateConfig(bytes.NewReader(asJSON))
	require.NoError(t, err)

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
	err = Import(ctx, logger, true, madminClient, minioClient, importConfig)
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
		err = Import(ctx, logger, false, madminClient, minioClient, importConfig)
		require.NoError(t, err)

		buckets, err = minioClient.ListBuckets(ctx)
		require.NoError(t, err)
		require.Len(t, buckets, len(bucketsToImport))
		require.Equal(t, bucketsToImport[0].Name, buckets[0].Name)

		quota, err := madminClient.GetBucketQuota(ctx, bucketsToImport[0].Name)
		require.NoError(t, err)
		require.Equal(t, uint64(bucketsToImport[0].Quota["Size"].(int)), quota.Size)

		lifecycle, err := minioClient.GetBucketLifecycle(ctx, bucketsToImport[0].Name)
		require.NoError(t, err)
		require.Len(t, lifecycle.Rules, 1)
		require.Equal(t, "rule1", lifecycle.Rules[0].ID)
		require.Equal(t, "Enabled", lifecycle.Rules[0].Status)
		require.Equal(t, 1, int(lifecycle.Rules[0].Expiration.Days))

		_, err = minioClient.GetBucketPolicy(ctx, bucketsToImport[0].Name)
		require.NoError(t, err)

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

	importConfig = ImportConfig{
		Users: []user{
			{
				AccessKey: "first",
				SecretKey: "secretnicewowwow",
				Policies:  []string{"this-policy-does-not-exist"},
			},
		},
	}
	err = Import(ctx, logger, false, madminClient, minioClient, importConfig)
	require.Error(t, err)

	importConfig = ImportConfig{
		Users: []user{
			{
				AccessKey: "missing-secret-key",
			},
		},
	}
	err = Import(ctx, logger, false, madminClient, minioClient, importConfig)
	require.Error(t, err)

	importConfig = ImportConfig{
		Buckets: []bucket{
			{
				Name: "!@#$%^&badnameשםלאטוב",
			},
		},
	}
	err = Import(ctx, logger, false, madminClient, minioClient, importConfig)
	require.Error(t, err)

	importConfig = ImportConfig{
		Buckets: []bucket{
			{
				Name: "bad-lifecycle",
				Policy: map[string]any{
					"not a valid policy": true,
				},
			},
		},
	}
	err = Import(ctx, logger, false, madminClient, minioClient, importConfig)
	require.Error(t, err)
}
