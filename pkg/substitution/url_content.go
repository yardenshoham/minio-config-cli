package substitution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// ErrUnsupportedURLScheme is returned by the url lookup when the URL uses a scheme other than http, https, or file.
var ErrUnsupportedURLScheme = errors.New("unsupported URL scheme")

type urlContentLookup struct {
	httpClient *http.Client
}

func (l *urlContentLookup) Lookup(ctx context.Context, rawURL []byte) ([]byte, error) {
	parsed, err := url.Parse(string(rawURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %q: %w", rawURL, err)
	}
	switch parsed.Scheme {
	case "http", "https":
		return l.fetchHTTP(ctx, string(rawURL))
	case "file":
		return new(fileLookup).Lookup(ctx, []byte(parsed.Path))
	default:
		return nil, fmt.Errorf("failed to fetch URL %q with unsupported scheme %q: %w", rawURL, parsed.Scheme, ErrUnsupportedURLScheme)
	}
}

func (l *urlContentLookup) fetchHTTP(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for URL %q: %w", rawURL, err)
	}
	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch URL %q: HTTP status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from URL %q: %w", rawURL, err)
	}
	return body, nil
}
