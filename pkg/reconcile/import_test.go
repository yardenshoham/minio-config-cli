package reconcile

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	miniotestcontainer "github.com/testcontainers/testcontainers-go/modules/minio"
)

func TestImport(t *testing.T) {
	// create minio container
	ctx := context.Background()
	minioContainer, err := miniotestcontainer.Run(ctx, "minio/minio:RELEASE.2025-02-03T21-03-04Z")
	defer func() {
		err := testcontainers.TerminateContainer(minioContainer)
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	endpoint, err := minioContainer.ConnectionString(ctx)
	assert.NoError(t, err)

	creds := credentials.NewStaticV4("minioadmin", "minioadmin", "")
	madminClient, err := madmin.NewWithOptions(endpoint, &madmin.Options{
		Secure: false,
		Creds:  creds,
	})
	assert.NoError(t, err)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Secure: false,
		Creds:  creds,
	})
	assert.NoError(t, err)

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
		},
	}
	ImportConfig := ImportConfig{
		Users:    usersToImport,
		Policies: policiesToImport,
		Buckets:  bucketsToImport,
	}
	users, err := madminClient.ListUsers(ctx)
	assert.NoError(t, err)
	assert.Len(t, users, 0)

	policies, err := madminClient.ListCannedPolicies(ctx)
	assert.NoError(t, err)
	builtinPinnedPoliciesAmount := len(policies)

	buckets, err := minioClient.ListBuckets(ctx)
	assert.NoError(t, err)
	assert.Len(t, buckets, 0)

	// twice to check idempotency
	for i := 0; i < 2; i++ {
		err = Import(logger, ctx, false, madminClient, minioClient, ImportConfig)
		assert.NoError(t, err)

		buckets, err = minioClient.ListBuckets(ctx)
		assert.NoError(t, err)
		assert.Len(t, buckets, len(bucketsToImport))
		assert.Equal(t, bucketsToImport[0].Name, buckets[0].Name)

		policies, err = madminClient.ListCannedPolicies(ctx)
		assert.NoError(t, err)
		assert.Len(t, policies, builtinPinnedPoliciesAmount+len(policiesToImport))
		assert.Contains(t, policies, readFoobarBucketPolicyName)

		users, err = madminClient.ListUsers(ctx)
		assert.NoError(t, err)
		assert.Len(t, users, len(usersToImport))
		assert.Contains(t, users, "first")
		assert.Contains(t, users, "second")
		assert.Equal(t, madmin.AccountEnabled, users["first"].Status)
		assert.Equal(t, madmin.AccountDisabled, users["second"].Status)
		assert.Equal(t, readFoobarBucketPolicyName, users["first"].PolicyName)
	}

	testdataConfigFile, err := os.Open("../../testdata/config.yaml")
	assert.NoError(t, err)
	defer testdataConfigFile.Close()

	testdataConfig, err := LoadConfig(testdataConfigFile)
	assert.NoError(t, err)
	err = Import(logger, ctx, false, madminClient, minioClient, *testdataConfig)
	assert.NoError(t, err)
}
