package substitution

import (
	"context"
	"fmt"
	"os"
)

type envLookup struct{}

func (l *envLookup) Lookup(_ context.Context, key []byte) ([]byte, error) {
	value, exists := os.LookupEnv(string(key))
	if !exists {
		return nil, fmt.Errorf("failed to resolve env var %q: not defined", key)
	}
	return []byte(value), nil
}
