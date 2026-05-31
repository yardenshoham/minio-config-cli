package cmd

import (
	"cmp"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/madmin-go/v4"
	"github.com/minio/minio-go/v7"
	"github.com/spf13/cobra"
	"github.com/yardenshoham/minio-config-cli/pkg/auth"
	"github.com/yardenshoham/minio-config-cli/pkg/reconciliation"
)

func newImportCmd() *cobra.Command {
	var (
		importFileLocations []string
		dryRun              bool
		cfg                 auth.Config
	)
	importCmd := &cobra.Command{
		Use:   "import MINIO_URL",
		Short: "Import configuration from the specified files",
		Example: `minio-config-cli import http://localhost:9000 \
    --access-key=minioadmin --secret-key=minioadmin \
    --import-file-location=config.yaml

minio-config-cli import https://minio.example.com \
    --oidc-issuer-url=https://keycloak.example.com/realms/minio \
    --oidc-client-id=minio-client \
    --oidc-client-secret=$OIDC_CLIENT_SECRET \
    --import-file-location=config.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			applyEnvFallback(&cfg)

			parsed, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("failed to parse minio url: %w", err)
			}
			if parsed.Host == "" {
				return fmt.Errorf("failed to parse minio url: missing host in %q", args[0])
			}
			secure := parsed.Scheme == "https"
			stsEndpoint := parsed.Scheme + "://" + parsed.Host

			creds, err := auth.BuildCredentials(cmd.Context(), stsEndpoint, cfg)
			if err != nil {
				return fmt.Errorf("failed to build credentials: %w", err)
			}
			madminClient, err := madmin.NewWithOptions(parsed.Host, &madmin.Options{
				Secure: secure,
				Creds:  creds,
			})
			if err != nil {
				return fmt.Errorf("failed to create madmin client: %w", err)
			}
			minioClient, err := minio.New(parsed.Host, &minio.Options{
				Creds:  creds,
				Secure: secure,
			})
			if err != nil {
				return fmt.Errorf("failed to create minio client: %w", err)
			}
			logger := slog.New(slog.NewTextHandler(cmd.OutOrStdout(), nil))
			if dryRun {
				logger.Info("running in dry-run mode")
				logger = logger.With("dry-run", "true")
			}
			filePaths := []string{}
			for _, importFileLocation := range importFileLocations {
				err := filepath.WalkDir(importFileLocation, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if d.Type().IsRegular() {
						filePaths = append(filePaths, path)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to walk import file locations: %w", err)
				}
			}
			ctx := cmd.Context()
			for _, path := range filePaths {
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", path, err)
				}
				config, err := reconciliation.LoadConfig(file)
				file.Close()
				if err != nil {
					return fmt.Errorf("failed to load config from file %s: %w", path, err)
				}
				err = reconciliation.Import(ctx, logger.With("file", path), dryRun, madminClient, minioClient, *config)
				if err != nil {
					return fmt.Errorf("failed to import from file %s: %w", path, err)
				}
			}
			return nil
		},
	}
	const importFileLocationsFlagName = "import-file-location"
	importCmd.Flags().StringSliceVarP(&importFileLocations, importFileLocationsFlagName, "i", []string{}, "Import configuration from the specified files")
	if err := importCmd.MarkFlagRequired(importFileLocationsFlagName); err != nil {
		panic(err)
	}
	if err := importCmd.MarkFlagFilename(importFileLocationsFlagName, "yaml", "yml", "json"); err != nil {
		panic(err)
	}
	importCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Don't actually modify resources in the MinIO server")

	// Static auth flags.
	importCmd.Flags().StringVar(&cfg.AccessKey, "access-key", "", "MinIO access key (env: MINIO_ACCESS_KEY)")
	importCmd.Flags().StringVar(&cfg.SecretKey, "secret-key", "", "MinIO secret key (env: MINIO_SECRET_KEY)")

	// OIDC auth flags.
	importCmd.Flags().StringVar(&cfg.OIDCIssuerURL, "oidc-issuer-url", "", "OIDC issuer URL, e.g. https://keycloak.example.com/realms/minio (env: OIDC_ISSUER_URL)")
	importCmd.Flags().StringVar(&cfg.OIDCClientID, "oidc-client-id", "", "OIDC client ID (env: OIDC_CLIENT_ID)")
	importCmd.Flags().StringVar(&cfg.OIDCClientSecret, "oidc-client-secret", "", "OIDC client secret (env: OIDC_CLIENT_SECRET)")
	importCmd.Flags().StringSliceVar(&cfg.OIDCExtraScopes, "oidc-extra-scope", nil, "Extra OIDC scopes added on top of 'openid' (env: OIDC_EXTRA_SCOPES, comma-separated)")
	importCmd.Flags().StringVar(&cfg.GrantType, "grant-type", "", "OIDC grant type: auto|password|client-credentials (env: OIDC_GRANT_TYPE)")
	importCmd.Flags().StringVar(&cfg.Username, "username", "", "Username for the password grant (env: OIDC_USERNAME)")
	importCmd.Flags().StringVar(&cfg.Password, "password", "", "Password for the password grant (env: OIDC_PASSWORD)")

	// Mode mixing is validated centrally in auth.BuildCredentials
	// (auth.ErrMixedModes), which also catches the env-var case that cobra's
	// MarkFlagsMutuallyExclusive cannot see.

	return importCmd
}

// applyEnvFallback fills empty fields of cfg from environment variables.
// Explicit flags always win because cfg has already been populated by cobra.
func applyEnvFallback(cfg *auth.Config) {
	cfg.AccessKey = cmp.Or(cfg.AccessKey, os.Getenv("MINIO_ACCESS_KEY"))
	cfg.SecretKey = cmp.Or(cfg.SecretKey, os.Getenv("MINIO_SECRET_KEY"))
	cfg.OIDCIssuerURL = cmp.Or(cfg.OIDCIssuerURL, os.Getenv("OIDC_ISSUER_URL"))
	cfg.OIDCClientID = cmp.Or(cfg.OIDCClientID, os.Getenv("OIDC_CLIENT_ID"))
	cfg.OIDCClientSecret = cmp.Or(cfg.OIDCClientSecret, os.Getenv("OIDC_CLIENT_SECRET"))
	cfg.GrantType = cmp.Or(cfg.GrantType, os.Getenv("OIDC_GRANT_TYPE"))
	cfg.Username = cmp.Or(cfg.Username, os.Getenv("OIDC_USERNAME"))
	cfg.Password = cmp.Or(cfg.Password, os.Getenv("OIDC_PASSWORD"))
	if len(cfg.OIDCExtraScopes) == 0 {
		cfg.OIDCExtraScopes = parseCSVEnv("OIDC_EXTRA_SCOPES")
	}
}

// parseCSVEnv returns the trimmed, non-empty comma-separated values of the
// given env var, or nil if the variable is empty/unset.
func parseCSVEnv(name string) []string {
	v := os.Getenv(name)
	if v == "" {
		return nil
	}
	var out []string
	for s := range strings.SplitSeq(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
