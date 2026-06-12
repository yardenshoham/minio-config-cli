package substitution

import (
	"bytes"
	"context"
	"fmt"
	"os"
)

type fileLookup struct{}

// Lookup reads the file at path and returns its content with a single
// trailing newline (LF or CRLF) removed, so the usual trailing newline in
// secret files does not corrupt the YAML document it is substituted into.
// Path resolution follows os.ReadFile semantics: relative paths are resolved
// against the process's current working directory.
func (l *fileLookup) Lookup(_ context.Context, path []byte) ([]byte, error) {
	content, err := os.ReadFile(string(path))
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}
	content = bytes.TrimSuffix(content, []byte("\n"))
	content = bytes.TrimSuffix(content, []byte("\r"))
	return content, nil
}
