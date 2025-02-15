package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	t.Parallel()
	rootCmd := newRootCmd()
	require.NoError(t, rootCmd.Execute())
}
