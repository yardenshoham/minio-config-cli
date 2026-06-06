package reconciliation

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	keycloak "github.com/stillya/testcontainers-keycloak"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	miniotestcontainer "github.com/testcontainers/testcontainers-go/modules/minio"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/yardenshoham/minio-config-cli/pkg/auth"
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

	t.Run("static", func(t *testing.T) {
		t.Parallel()
		ctx, madminClient, minioClient, logger, minioContainer := testSetup(t)
		defer func() {
			err := testcontainers.TerminateContainer(minioContainer)
			require.NoError(t, err)
		}()
		runImportScenario(ctx, t, madminClient, minioClient, logger)
	})

	kc := setupKeycloak(t)
	endpoint := kc.startMinIO(t)

	// The two OIDC subtests share the same MinIO: client-credentials drives the
	// full reconciliation scenario, password only smoke-tests that STS accepts
	// the JWT minted via the password grant. Running ServerInfo concurrently
	// with the scenario is safe — it never mutates state.
	t.Run("oidc/client-credentials", func(t *testing.T) {
		t.Parallel()
		ctx, madminClient, minioClient, logger := kc.clientsFor(t, endpoint, auth.Config{
			OIDCIssuerURL:    kc.issuerURL,
			OIDCClientID:     "minio-client",
			OIDCClientSecret: "test-client-secret",
			GrantType:        auth.GrantClientCredentials,
		})
		runImportScenario(ctx, t, madminClient, minioClient, logger)
	})

	t.Run("oidc/password", func(t *testing.T) {
		t.Parallel()
		ctx, madminClient, _, _ := kc.clientsFor(t, endpoint, auth.Config{
			OIDCIssuerURL:    kc.issuerURL,
			OIDCClientID:     "minio-client",
			OIDCClientSecret: "test-client-secret",
			GrantType:        auth.GrantPassword,
			Username:         "testuser",
			Password:         "testpassword",
		})
		_, err := madminClient.ServerInfo(ctx)
		require.NoError(t, err)
	})
}

func runImportScenario(ctx context.Context, t *testing.T, madminClient *madmin.AdminClient, minioClient *minio.Client, logger *slog.Logger) {
	t.Helper()
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
		{
			Name: "versioned-bucket",
			Versioning: map[string]any{
				"Status": "Enabled",
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

		versioningConfig, err := minioClient.GetBucketVersioning(ctx, bucketsToImport[1].Name)
		require.NoError(t, err)
		require.True(t, versioningConfig.Enabled())

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

	testdataConfig, err := LoadConfig(ctx, testdataConfigFile)
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

// keycloakEnv holds the shared Keycloak + docker-network handles used by the
// OIDC subtests of TestImport. Each subtest spins up its own MinIO container
// so the import-scenario assertions can run against a fresh server.
type keycloakEnv struct {
	nw         *testcontainers.DockerNetwork
	kcTokenURL string // host-mapped, e.g. http://127.0.0.1:54321/realms/minio/protocol/openid-connect/token
	issuerURL  string // what MinIO trusts: http://keycloak:8080/realms/minio
}

// setupKeycloak boots a Keycloak container with the test realm imported and
// a shared docker network. Both are cleaned up via t.Cleanup.
func setupKeycloak(t *testing.T) *keycloakEnv {
	t.Helper()
	ctx := t.Context()

	nw, err := tcnetwork.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nw.Remove(ctx) })

	kc, err := keycloak.Run(ctx, "quay.io/keycloak/keycloak:26.6.3",
		tcnetwork.WithNetwork([]string{"keycloak"}, nw),
		testcontainers.WithEnv(map[string]string{
			"KC_HOSTNAME":        "http://keycloak:8080",
			"KC_HOSTNAME_STRICT": "false",
			"KC_HTTP_ENABLED":    "true",
		}),
		keycloak.WithRealmImportFile("../../testdata/minio-realm.json"),
		keycloak.WithAdminUsername("admin"),
		keycloak.WithAdminPassword("admin"),
		testcontainers.WithWaitStrategy(
			wait.ForHTTP("/realms/minio/.well-known/openid-configuration").
				WithPort("8080/tcp").
				WithStartupTimeout(3*time.Minute),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(kc) })

	kcURL, err := kc.GetAuthServerURL(ctx)
	require.NoError(t, err)

	return &keycloakEnv{
		nw:         nw,
		kcTokenURL: kcURL + "/realms/minio/protocol/openid-connect/token",
		issuerURL:  "http://keycloak:8080/realms/minio",
	}
}

// startMinIO launches an OIDC-configured MinIO on the shared docker network
// and returns its host-mapped endpoint. The container is registered for
// cleanup against t.
func (e *keycloakEnv) startMinIO(t *testing.T) string {
	t.Helper()
	ctx := t.Context()

	mc, err := miniotestcontainer.Run(ctx, "coollabsio/minio:RELEASE.2025-10-15T17-29-55Z",
		tcnetwork.WithNetwork(nil, e.nw),
		testcontainers.WithEnv(map[string]string{
			"MINIO_IDENTITY_OPENID_CONFIG_URL":    "http://keycloak:8080/realms/minio/.well-known/openid-configuration",
			"MINIO_IDENTITY_OPENID_CLIENT_ID":     "minio-client",
			"MINIO_IDENTITY_OPENID_CLIENT_SECRET": "test-client-secret",
			"MINIO_IDENTITY_OPENID_SCOPES":        "openid,minio",
		}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(mc) })

	endpoint, err := mc.ConnectionString(ctx)
	require.NoError(t, err)
	return endpoint
}

// clientsFor builds madmin/minio clients backed by STS web identity
// credentials minted from cfg, targeting the MinIO at endpoint.
func (e *keycloakEnv) clientsFor(t *testing.T, endpoint string, cfg auth.Config) (context.Context, *madmin.AdminClient, *minio.Client, *slog.Logger) {
	t.Helper()
	ctx := t.Context()

	creds, err := auth.BuildCredentials(ctx, "http://"+endpoint, cfg, auth.WithTokenURL(e.kcTokenURL))
	require.NoError(t, err)

	madminClient, err := madmin.NewWithOptions(endpoint, &madmin.Options{Secure: false, Creds: creds})
	require.NoError(t, err)
	minioClient, err := minio.New(endpoint, &minio.Options{Secure: false, Creds: creds})
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	return ctx, madminClient, minioClient, logger
}

func TestLoadConfigVariableSubstitution(t *testing.T) {
	t.Setenv("_TEST_LOADCONFIG_BUCKET", "substituted-bucket")

	cfg, err := LoadConfig(t.Context(), strings.NewReader("buckets:\n  - name: $(env:_TEST_LOADCONFIG_BUCKET)\n"))
	require.NoError(t, err)
	require.Len(t, cfg.Buckets, 1)
	require.Equal(t, "substituted-bucket", cfg.Buckets[0].Name)
}
