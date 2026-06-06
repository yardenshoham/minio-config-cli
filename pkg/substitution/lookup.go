package substitution

import "context"

// Lookup resolves a variable value for a given key.
type Lookup interface {
	Lookup(ctx context.Context, key []byte) ([]byte, error)
}
