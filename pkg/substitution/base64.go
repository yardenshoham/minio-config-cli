package substitution

import (
	"context"
	"encoding/base64"
	"fmt"
)

type base64DecoderLookup struct{}

func (l *base64DecoderLookup) Lookup(_ context.Context, key []byte) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(string(key))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 value %q: %w", key, err)
	}
	return decoded, nil
}

type base64EncoderLookup struct{}

func (l *base64EncoderLookup) Lookup(_ context.Context, key []byte) ([]byte, error) {
	return base64.StdEncoding.AppendEncode(nil, key), nil
}
