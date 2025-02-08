package validation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	file, err := os.Open("../../testdata/config.yaml")
	assert.NoError(t, err)
	defer file.Close()
	err = ValidateConfig(file)
	assert.NoError(t, err)
}
