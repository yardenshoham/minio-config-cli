package auth

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeOIDC impersonates an OIDC discovery + token endpoint and records the
// last token request so tests can assert what grant was used.
type fakeOIDC struct {
	srv          *httptest.Server
	lastForm     url.Values
	lastAuthUser string
	lastAuthPass string
}

func newFakeOIDC(t *testing.T) *fakeOIDC {
	t.Helper()
	f := &fakeOIDC{}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token_endpoint": f.srv.URL + "/token"})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.lastForm = r.PostForm
		f.lastAuthUser, f.lastAuthPass, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fake-jwt",
			"token_type":   "Bearer",
			"expires_in":   300,
		})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func TestBuildCredentials_Static(t *testing.T) {
	t.Parallel()
	creds, err := BuildCredentials(t.Context(), "http://minio.example.com", Config{
		AccessKey: "ak",
		SecretKey: "sk",
	})
	require.NoError(t, err)
	// GetWithContext takes *credentials.CredContext (not context.Context);
	// nil is documented as accepted.
	v, err := creds.GetWithContext(nil)
	require.NoError(t, err)
	require.Equal(t, "ak", v.AccessKeyID)
	require.Equal(t, "sk", v.SecretAccessKey)
}

func TestBuildCredentials_ConfigErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{"mixed modes", Config{AccessKey: "ak", SecretKey: "sk", OIDCIssuerURL: "x"}, ErrMixedModes},
		{"no mode", Config{}, ErrNoCredentials},
		{"static missing secret", Config{AccessKey: "ak"}, ErrIncompleteConfig},
		{"oidc missing client id", Config{OIDCIssuerURL: "x"}, ErrIncompleteConfig},
		{"password grant missing username", Config{OIDCIssuerURL: "x", OIDCClientID: "c", GrantType: GrantPassword}, ErrIncompleteConfig},
		{"client-credentials missing secret", Config{OIDCIssuerURL: "x", OIDCClientID: "c", GrantType: GrantClientCredentials}, ErrIncompleteConfig},
		{"unknown grant", Config{OIDCIssuerURL: "x", OIDCClientID: "c", GrantType: "magic"}, ErrUnknownGrant},
		// A stray OIDC env var alone (e.g. OIDC_CLIENT_SECRET) is enough to
		// flip into OIDC mode; it should fail with an incomplete-config error,
		// not the "no credentials" one.
		{"oidc client secret only", Config{OIDCClientSecret: "s"}, ErrIncompleteConfig},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := BuildCredentials(t.Context(), "http://minio.example.com", tc.cfg)
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestBuildCredentials_OIDCFlows exercises the OIDC grant flows by invoking
// the token-fetching closure through credentials.GetWithContext. The STS
// step is expected to fail (httptest server doesn't speak STS XML); we only
// care that the correct OAuth2 request shape was issued.
func TestBuildCredentials_OIDCFlows(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		cfg       Config
		wantGrant string
		wantForm  map[string]string
		wantBasic [2]string // user, pass (skipped if user is "")
	}{
		{
			name: "client-credentials with discovery",
			cfg: Config{
				OIDCClientID:     "client",
				OIDCClientSecret: "secret",
				OIDCExtraScopes:  []string{"minio"},
			},
			wantGrant: "client_credentials",
			wantForm:  map[string]string{"scope": "openid minio"},
			wantBasic: [2]string{"client", "secret"},
		},
		{
			name: "password grant",
			cfg: Config{
				OIDCClientID:     "client",
				OIDCClientSecret: "secret",
				GrantType:        GrantPassword,
				Username:         "alice",
				Password:         "wonderland",
			},
			wantGrant: "password",
			wantForm:  map[string]string{"username": "alice", "password": "wonderland"},
		},
		{
			name: "auto selects password when username given",
			cfg: Config{
				OIDCClientID:     "client",
				OIDCClientSecret: "secret",
				Username:         "alice",
				Password:         "wonderland",
				GrantType:        GrantAuto,
			},
			wantGrant: "password",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newFakeOIDC(t)
			cfg := tc.cfg
			cfg.OIDCIssuerURL = f.srv.URL
			creds, err := BuildCredentials(t.Context(), f.srv.URL, cfg)
			require.NoError(t, err)
			_, _ = creds.GetWithContext(nil)
			require.Equal(t, tc.wantGrant, f.lastForm.Get("grant_type"))
			for k, v := range tc.wantForm {
				require.Equal(t, v, f.lastForm.Get(k), "form field %q", k)
			}
			if tc.wantBasic[0] != "" {
				require.Equal(t, tc.wantBasic[0], f.lastAuthUser)
				require.Equal(t, tc.wantBasic[1], f.lastAuthPass)
			}
		})
	}
}

func TestBuildCredentials_WithTokenURLSkipsDiscovery(t *testing.T) {
	t.Parallel()
	f := newFakeOIDC(t)
	// Issuer URL points at a black hole; WithTokenURL must skip discovery.
	creds, err := BuildCredentials(t.Context(), f.srv.URL, Config{
		OIDCIssuerURL:    "http://127.0.0.1:1",
		OIDCClientID:     "client",
		OIDCClientSecret: "secret",
	}, WithTokenURL(f.srv.URL+"/token"))
	require.NoError(t, err)
	_, _ = creds.GetWithContext(nil)
	require.Equal(t, "client_credentials", f.lastForm.Get("grant_type"))
}

func TestDiscoverTokenEndpoint_Error(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(http.NotFound))
	t.Cleanup(srv.Close)
	// Discovery retries until ctx cancellation; bound it with a short
	// deadline so the test fails fast if retry stops respecting ctx.
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := discoverTokenEndpoint(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), srv.URL)
	require.ErrorIs(t, err, ErrDiscovery)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
