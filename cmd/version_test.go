package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	t.Parallel()
	versionCmd := newVersionCmd()
	var stdout bytes.Buffer
	versionCmd.SetOut(&stdout)
	require.NoError(t, versionCmd.Execute())

	// it should unmarshal well
	var version versionInfo
	err := json.Unmarshal(stdout.Bytes(), &version)
	require.NoError(t, err)
	require.NotEmpty(t, version.Version)
	require.NotEmpty(t, version.GoVersion)
}
