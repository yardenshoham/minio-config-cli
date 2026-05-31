// Package auth builds *credentials.Credentials values for the MinIO
// clients used by minio-config-cli. It supports both static MinIO
// access-key/secret-key credentials and OIDC-backed STS credentials
// (AssumeRoleWithWebIdentity) using either the client-credentials or
// resource-owner password grant.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Grant types accepted in Config.GrantType.
const (
	GrantAuto              = "auto"
	GrantPassword          = "password"
	GrantClientCredentials = "client-credentials"
)

// Sentinel errors returned (wrapped) by BuildCredentials so callers can
// classify configuration problems without string matching.
var (
	// ErrMixedModes is returned when both static and OIDC fields are set.
	ErrMixedModes = errors.New("auth: cannot mix static and OIDC credentials")
	// ErrNoCredentials is returned when no authentication mode is configured.
	ErrNoCredentials = errors.New("auth: no credentials configured")
	// ErrIncompleteConfig is returned when a chosen mode is missing required fields.
	ErrIncompleteConfig = errors.New("auth: incomplete configuration")
	// ErrUnknownGrant is returned for an unrecognised Config.GrantType value.
	ErrUnknownGrant = errors.New("auth: unknown grant type")
	// ErrDiscovery is returned when OIDC issuer discovery fails.
	ErrDiscovery = errors.New("auth: OIDC discovery failed")
)

// Config describes how to build credentials. Either the static fields
// (AccessKey/SecretKey) or the OIDC fields must be populated, not both.
type Config struct {
	// Static mode.
	AccessKey string
	SecretKey string

	// OIDC mode.
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCExtraScopes  []string
	GrantType        string
	Username         string
	Password         string
}

// IsStatic reports whether any static-mode field is set.
func (c Config) IsStatic() bool {
	return c.AccessKey != "" || c.SecretKey != ""
}

// IsOIDC reports whether any OIDC-mode field is set.
func (c Config) IsOIDC() bool {
	return c.OIDCIssuerURL != "" ||
		c.OIDCClientID != "" ||
		c.OIDCClientSecret != "" ||
		len(c.OIDCExtraScopes) > 0 ||
		(c.GrantType != "" && c.GrantType != GrantAuto) ||
		c.Username != "" ||
		c.Password != ""
}

// Option mutates BuildCredentials' internal options.
type Option func(*options)

type options struct {
	tokenURL string
}

// WithTokenURL bypasses OIDC discovery and tells BuildCredentials to POST
// the grant request to the given URL directly. The Config.OIDCIssuerURL is
// still required for validation but no HTTP request is made to it. This is
// used by integration tests and by deployments where /.well-known is
// unreachable from the caller.
func WithTokenURL(u string) Option {
	return func(o *options) { o.tokenURL = u }
}

// BuildCredentials returns *credentials.Credentials suitable for both
// minio-go and madmin-go. stsEndpoint must be a full URL with scheme
// (e.g. "https://minio.example.com"); it is ignored in static mode.
func BuildCredentials(ctx context.Context, stsEndpoint string, cfg Config, opts ...Option) (*credentials.Credentials, error) {
	if cfg.IsStatic() && cfg.IsOIDC() {
		return nil, ErrMixedModes
	}
	if cfg.IsStatic() {
		if cfg.AccessKey == "" || cfg.SecretKey == "" {
			return nil, fmt.Errorf("%w: static mode requires both access key and secret key", ErrIncompleteConfig)
		}
		return credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""), nil
	}
	if !cfg.IsOIDC() {
		return nil, fmt.Errorf("%w (set access key/secret key or OIDC flags)", ErrNoCredentials)
	}

	grant, err := resolveGrant(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve grant type: %w", err)
	}
	if err := validateOIDC(cfg, grant); err != nil {
		return nil, fmt.Errorf("validate OIDC config: %w", err)
	}

	o := options{}
	for _, apply := range opts {
		apply(&o)
	}

	tokenURL := o.tokenURL
	if tokenURL == "" {
		tokenURL, err = discoverTokenEndpoint(ctx, cfg.OIDCIssuerURL)
		if err != nil {
			return nil, fmt.Errorf("discover OIDC token endpoint: %w", err)
		}
	}

	scopes := append([]string{"openid"}, cfg.OIDCExtraScopes...)
	// The closure captures ctx from BuildCredentials. That is fine for the
	// minio-config-cli CLI (its cmd.Context lives for the whole process), but
	// callers that want STS-credential refresh to outlive the build call
	// should pass a long-lived context here.
	fetchToken := newFetchToken(ctx, cfg, grant, tokenURL, scopes)

	creds, err := credentials.NewSTSWebIdentity(stsEndpoint, fetchToken)
	if err != nil {
		return nil, fmt.Errorf("build STS web identity credentials: %w", err)
	}
	return creds, nil
}

func resolveGrant(cfg Config) (string, error) {
	switch cfg.GrantType {
	case "", GrantAuto:
		if cfg.Username != "" && cfg.Password != "" {
			return GrantPassword, nil
		}
		return GrantClientCredentials, nil
	case GrantPassword, GrantClientCredentials:
		return cfg.GrantType, nil
	default:
		return "", fmt.Errorf("%w: %q (expected %q, %q, or %q)",
			ErrUnknownGrant, cfg.GrantType, GrantAuto, GrantPassword, GrantClientCredentials)
	}
}

func validateOIDC(cfg Config, grant string) error {
	if cfg.OIDCIssuerURL == "" {
		return fmt.Errorf("%w: OIDC mode requires --oidc-issuer-url", ErrIncompleteConfig)
	}
	if cfg.OIDCClientID == "" {
		return fmt.Errorf("%w: OIDC mode requires --oidc-client-id", ErrIncompleteConfig)
	}
	switch grant {
	case GrantPassword:
		if cfg.Username == "" || cfg.Password == "" {
			return fmt.Errorf("%w: password grant requires --username and --password", ErrIncompleteConfig)
		}
	case GrantClientCredentials:
		if cfg.OIDCClientSecret == "" {
			return fmt.Errorf("%w: client-credentials grant requires --oidc-client-secret", ErrIncompleteConfig)
		}
	}
	return nil
}

func newFetchToken(ctx context.Context, cfg Config, grant, tokenURL string, scopes []string) func() (*credentials.WebIdentityToken, error) {
	return func() (*credentials.WebIdentityToken, error) {
		var (
			tok *oauth2.Token
			err error
		)
		switch grant {
		case GrantClientCredentials:
			cc := clientcredentials.Config{
				ClientID:     cfg.OIDCClientID,
				ClientSecret: cfg.OIDCClientSecret,
				TokenURL:     tokenURL,
				Scopes:       scopes,
			}
			tok, err = cc.Token(ctx)
		case GrantPassword:
			oc := oauth2.Config{
				ClientID:     cfg.OIDCClientID,
				ClientSecret: cfg.OIDCClientSecret,
				Endpoint:     oauth2.Endpoint{TokenURL: tokenURL},
				Scopes:       scopes,
			}
			tok, err = oc.PasswordCredentialsToken(ctx, cfg.Username, cfg.Password)
		default:
			return nil, fmt.Errorf("unsupported grant type %q", grant)
		}
		if err != nil {
			return nil, fmt.Errorf("fetch OIDC token: %w", err)
		}
		// Expiry intentionally left at zero: see PLAN §9 finding 2.
		return &credentials.WebIdentityToken{Token: tok.AccessToken}, nil
	}
}

func discoverTokenEndpoint(ctx context.Context, issuer string) (string, error) {
	discURL := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discURL, nil)
	if err != nil {
		return "", fmt.Errorf("%w: build request: %w", ErrDiscovery, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDiscovery, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: status %d", ErrDiscovery, resp.StatusCode)
	}
	var d discoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return "", fmt.Errorf("%w: decode response: %w", ErrDiscovery, err)
	}
	if d.TokenEndpoint == "" {
		return "", fmt.Errorf("%w: empty token_endpoint", ErrDiscovery)
	}
	return d.TokenEndpoint, nil
}

// discoveryDocument is a partial representation of the OpenID Connect
// discovery document (RFC 8414 / OIDC Discovery 1.0). Only the fields we
// consume are decoded.
type discoveryDocument struct {
	TokenEndpoint string `json:"token_endpoint"`
}
