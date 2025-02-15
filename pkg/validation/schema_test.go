package validation

import (
	"bytes"
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

func TestValidateConfigInvalid(t *testing.T) {
	t.Parallel()
	config := "I am not valid YAML"
	err := ValidateConfig(bytes.NewReader([]byte(config)))
	require.Error(t, err)
}
