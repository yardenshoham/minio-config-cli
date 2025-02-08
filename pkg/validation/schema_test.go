package validation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()
	file, err := os.Open("../../testdata/config.yaml")
	require.NoError(t, err)
	defer file.Close()
	err = ValidateConfig(file)
	require.NoError(t, err)
}
