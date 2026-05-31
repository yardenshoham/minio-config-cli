package cmd

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yardenshoham/minio-config-cli/pkg/auth"
)

func TestImportCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing args",
			args: []string{},
		},
		{
			name: "invalid minio url",
			args: []string{
				"://bad",
				"--access-key=minioadmin",
				"--secret-key=minioadmin",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "no credentials configured",
			args: []string{
				"http://localhost:9000",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "static missing secret key",
			args: []string{
				"http://localhost:9000",
				"--access-key=minioadmin",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "static and oidc flags mixed",
			args: []string{
				"http://localhost:9000",
				"--access-key=minioadmin",
				"--secret-key=minioadmin",
				"--oidc-issuer-url=https://issuer.example.com",
				"--oidc-client-id=client",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "oidc missing client id",
			args: []string{
				"http://localhost:9000",
				"--oidc-issuer-url=https://issuer.example.com",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "password grant missing username",
			args: []string{
				"http://localhost:9000",
				"--oidc-issuer-url=https://issuer.example.com",
				"--oidc-client-id=client",
				"--grant-type=password",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "client-credentials grant missing client secret",
			args: []string{
				"http://localhost:9000",
				"--oidc-issuer-url=https://issuer.example.com",
				"--oidc-client-id=client",
				"--grant-type=client-credentials",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "non-existent file",
			args: []string{
				"http://localhost:9000",
				"--access-key=minioadmin",
				"--secret-key=minioadmin",
				"--import-file-location=doesnotexistiamsure.yaml",
			},
		},
		{
			name: "malformed config",
			args: []string{
				"http://localhost:9000",
				"--access-key=minioadmin",
				"--secret-key=minioadmin",
				"--import-file-location=../testdata/malformed/config.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			importCmd := newImportCmd()
			importCmd.SetArgs(tt.args)
			require.Error(t, importCmd.Execute())
		})
	}
}

// TestImportCmd_EnvVarFallback verifies that MINIO_ACCESS_KEY and
// MINIO_SECRET_KEY env vars are picked up when no --access-key/--secret-key
// flags are passed. We use a nonexistent import path so the test stays
// offline: with env fallback working, validation passes and the command
// fails when it tries to open the file (fs.ErrNotExist).
func TestImportCmd_EnvVarFallback(t *testing.T) {
	t.Setenv("MINIO_ACCESS_KEY", "ak")
	t.Setenv("MINIO_SECRET_KEY", "sk")

	importCmd := newImportCmd()
	importCmd.SetArgs([]string{
		"http://localhost:9000",
		"--import-file-location=doesnotexistiamsure.yaml",
	})
	err := importCmd.Execute()
	require.Error(t, err)
	require.NotErrorIs(t, err, auth.ErrNoCredentials)
	require.NotErrorIs(t, err, auth.ErrIncompleteConfig)
	require.ErrorIs(t, err, fs.ErrNotExist)
}
