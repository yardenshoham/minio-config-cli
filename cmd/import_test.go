package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
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
				"invalid-url",
				"minioadmin",
				"minioadmin",
				"--import-file-location=../testdata/config.yaml",
			},
		},
		{
			name: "non-existent file",
			args: []string{
				"http://localhost:9000",
				"minioadmin",
				"minioadmin",
				"--import-file-location=doesnotexistiamsure.yaml",
			},
		},
		{
			name: "malformed config",
			args: []string{
				"http://localhost:9000",
				"minioadmin",
				"minioadmin",
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
