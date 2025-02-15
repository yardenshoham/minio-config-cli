package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd(t *testing.T) {
	t.Parallel()
	versionCmd := newVersionCmd()
	require.NoError(t, versionCmd.Execute())
}
