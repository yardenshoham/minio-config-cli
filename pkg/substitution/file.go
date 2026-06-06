package substitution

import (
	"context"
	"fmt"
	"os"
)

type fileLookup struct{}

// Lookup reads the file at path and returns its content.
// Path resolution follows os.ReadFile semantics: relative paths are resolved
// against the process's current working directory.
func (l *fileLookup) Lookup(_ context.Context, path []byte) ([]byte, error) {
	content, err := os.ReadFile(string(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return content, nil
}
