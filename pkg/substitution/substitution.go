package substitution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

// innermostPattern matches $(prefix:key) where the value contains no $, (, or ).
// This identifies the innermost (deepest) variable reference, enabling inside-out resolution.
var innermostPattern = regexp.MustCompile(`\$\(([^$()]+)\)`)

// escapePrefix and escapePlaceholder are used to protect $$( sequences during substitution.
var (
	escapePrefix      = []byte("$$(")
	escapePlaceholder = []byte("\x00ESCAPED_VAR_PREFIX\x00")
	varPrefix         = []byte("$(")
)

// ErrUnknownPrefix is returned by Substitute when a variable references a prefix that is not registered.
var ErrUnknownPrefix = errors.New("unknown variable prefix")

// Option configures the Substitute function.
type Option func(*substituter)

// WithHTTPClient injects a custom HTTP client used by the url lookup.
// Useful for testing with httptest.NewServer.
func WithHTTPClient(client *http.Client) Option {
	return func(s *substituter) {
		s.httpClient = client
	}
}

type substituter struct {
	httpClient *http.Client
}

// Substitute replaces all $(prefix:key) occurrences in input with their resolved values.
// Substitution proceeds inside-out: the innermost variable is resolved first, which
// allows nested expressions such as $(file:$(env:CONFIG_PATH)).
//
// To include a literal $(prefix:key) in the output without substitution, escape it
// as $$(prefix:key). All other text is passed through unchanged.
func Substitute(ctx context.Context, input []byte, opts ...Option) ([]byte, error) {
	s := &substituter{
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s.substitute(ctx, input)
}

func (s *substituter) substitute(ctx context.Context, input []byte) ([]byte, error) {
	// maxDepth bounds the number of passes, i.e. the nesting depth of
	// expressions. Each pass resolves every innermost variable in the input,
	// so the number of variables in a config is unbounded.
	const maxDepth = 50
	registry := s.buildRegistry()

	// Replace $$( escape sequences with a placeholder to protect them from substitution.
	working := bytes.ReplaceAll(input, escapePrefix, escapePlaceholder)

	for range maxDepth {
		matches := innermostPattern.FindAllSubmatchIndex(working, -1)
		if matches == nil {
			break
		}
		var result bytes.Buffer
		last := 0
		for _, match := range matches {
			inner := working[match[2]:match[3]]
			resolved, err := s.resolve(ctx, inner, registry)
			if err != nil {
				return nil, fmt.Errorf("failed to substitute variable %q: %w", string(inner), err)
			}
			result.Write(working[last:match[0]])
			result.Write(resolved)
			last = match[1]
		}
		result.Write(working[last:])
		working = result.Bytes()
	}

	// If patterns remain after reaching the depth limit, report an error.
	if innermostPattern.Match(working) {
		return nil, fmt.Errorf("failed to substitute variables: maximum nesting depth reached, possible circular reference")
	}

	// Restore escaped sequences to their literal form.
	return bytes.ReplaceAll(working, escapePlaceholder, varPrefix), nil
}

func (s *substituter) buildRegistry() map[string]Lookup {
	return map[string]Lookup{
		"base64Decoder": &base64DecoderLookup{},
		"base64Encoder": &base64EncoderLookup{},
		"env":           &envLookup{},
		"file":          &fileLookup{},
		"urlDecoder":    &urlDecoderLookup{},
		"urlEncoder":    &urlEncoderLookup{},
		"url":           &urlContentLookup{httpClient: s.httpClient},
	}
}

func (s *substituter) resolve(ctx context.Context, inner []byte, registry map[string]Lookup) ([]byte, error) {
	before, after, ok := bytes.Cut(inner, []byte{':'})
	if !ok {
		return nil, fmt.Errorf("missing prefix separator ':' in variable %q", string(inner))
	}
	prefix := string(before)

	lookup, ok := registry[prefix]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownPrefix, prefix)
	}
	return lookup.Lookup(ctx, after)
}
