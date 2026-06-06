package substitution

import (
	"context"
	"fmt"
	"net/url"
)

type urlDecoderLookup struct{}

func (l *urlDecoderLookup) Lookup(_ context.Context, key []byte) ([]byte, error) {
	decoded, err := url.QueryUnescape(string(key))
	if err != nil {
		return nil, fmt.Errorf("failed to decode URL-encoded string %q: %w", key, err)
	}
	return []byte(decoded), nil
}

type urlEncoderLookup struct{}

func (l *urlEncoderLookup) Lookup(_ context.Context, key []byte) ([]byte, error) {
	return []byte(url.QueryEscape(string(key))), nil
}
