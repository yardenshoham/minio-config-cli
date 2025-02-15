package reconciliation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapAnyToByteSlice(t *testing.T) {
	t.Parallel()
	// create a map with a that can't be marshalled to JSON
	m := map[string]any{
		"a": func() {},
	}
	_, err := mapAnyToByteSlice(m)
	require.Error(t, err)
}
