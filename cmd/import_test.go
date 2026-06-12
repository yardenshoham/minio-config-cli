package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
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

func TestCollectConfigFiles(t *testing.T) {
	t.Parallel()

	t.Run("regular file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		require.NoError(t, os.WriteFile(path, []byte("{}"), 0600))
		files, err := collectConfigFiles([]string{path})
		require.NoError(t, err)
		require.Equal(t, []string{path}, files)
	})

	t.Run("symlinked file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		target := filepath.Join(dir, "real.yaml")
		require.NoError(t, os.WriteFile(target, []byte("{}"), 0600))
		link := filepath.Join(dir, "link.yaml")
		require.NoError(t, os.Symlink(target, link))
		files, err := collectConfigFiles([]string{link})
		require.NoError(t, err)
		require.Equal(t, []string{target}, files)
	})

	// Kubernetes ConfigMap/Secret mounts expose each file as a symlink into
	// a hidden ..data directory. The walk must import each file exactly once.
	t.Run("configmap-style mount", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		dataDir := filepath.Join(dir, "..2026_06_12_00_00_00.123")
		require.NoError(t, os.Mkdir(dataDir, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(dataDir, "config.yaml"), []byte("{}"), 0600))
		require.NoError(t, os.Symlink(dataDir, filepath.Join(dir, "..data")))
		require.NoError(t, os.Symlink(filepath.Join("..data", "config.yaml"), filepath.Join(dir, "config.yaml")))
		files, err := collectConfigFiles([]string{dir})
		require.NoError(t, err)
		require.Equal(t, []string{filepath.Join(dir, "config.yaml")}, files)
	})

	t.Run("no files found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		_, err := collectConfigFiles([]string{dir})
		require.ErrorContains(t, err, "no config files found")
	})

	t.Run("missing location", func(t *testing.T) {
		t.Parallel()
		_, err := collectConfigFiles([]string{filepath.Join(t.TempDir(), "missing.yaml")})
		require.ErrorIs(t, err, fs.ErrNotExist)
	})
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
