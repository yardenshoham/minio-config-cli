package substitution

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubstitute(t *testing.T) {
	// Stateless transformations: input → expected output, no external dependencies.
	t.Run("transforms", func(t *testing.T) {
		t.Parallel()
		cases := []struct{ name, input, want string }{
			{"no patterns", "no variables here", "no variables here"},
			{"empty", "", ""},
			{"base64Decoder", "$(base64Decoder:SGVsbG8=)", "Hello"},
			{"base64Encoder", "$(base64Encoder:Hello)", "SGVsbG8="},
			{"urlDecoder", "$(urlDecoder:Hello%20World)", "Hello World"},
			{"urlEncoder", "$(urlEncoder:Hello World)", "Hello+World"},
			{"escape", "$$(env:HOME)", "$(env:HOME)"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				result, err := Substitute(t.Context(), []byte(tc.input))
				require.NoError(t, err)
				require.Equal(t, tc.want, string(result))
			})
		}
	})

	// Error cases that require no external setup.
	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name        string
			input       string
			errContains string
			errIs       error
		}{
			{"env/undefined", "$(env:_SUBST_SURELY_NOT_SET_XYZ_12345)", "_SUBST_SURELY_NOT_SET_XYZ_12345", nil},
			{"base64Decoder/invalid", "$(base64Decoder:!!!notbase64!!!)", "failed to decode base64", nil},
			{"file/not found", "$(file:/nonexistent/path/file.txt)", "", os.ErrNotExist},
			{"url/unsupported scheme", "$(url:ftp://example.com)", "", ErrUnsupportedURLScheme},
			{"unknown prefix", "$(unknown:value)", "", ErrUnknownPrefix},
			{"missing colon", "$(noseparator)", "missing prefix separator", nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := Substitute(t.Context(), []byte(tc.input))
				if tc.errIs != nil {
					require.ErrorIs(t, err, tc.errIs)
				} else {
					require.ErrorContains(t, err, tc.errContains)
				}
			})
		}
	})

	t.Run("env/found", func(t *testing.T) {
		t.Setenv("_TEST_SUBST_VAR", "hello")
		result, err := Substitute(t.Context(), []byte("$(env:_TEST_SUBST_VAR)"))
		require.NoError(t, err)
		require.Equal(t, "hello", string(result))
	})

	t.Run("multiple substitutions", func(t *testing.T) {
		t.Setenv("_TEST_BUCKET_A", "foo")
		t.Setenv("_TEST_BUCKET_B", "bar")
		result, err := Substitute(t.Context(), []byte("$(env:_TEST_BUCKET_A)-$(env:_TEST_BUCKET_B)"))
		require.NoError(t, err)
		require.Equal(t, "foo-bar", string(result))
	})

	t.Run("escape alongside substitution", func(t *testing.T) {
		t.Setenv("_TEST_ESCAPE_VAR", "resolved")
		result, err := Substitute(t.Context(), []byte("$(env:_TEST_ESCAPE_VAR) and $$(env:_TEST_ESCAPE_VAR)"))
		require.NoError(t, err)
		require.Equal(t, "resolved and $(env:_TEST_ESCAPE_VAR)", string(result))
	})

	t.Run("file/found", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("file content"), 0600))
		result, err := Substitute(t.Context(), fmt.Appendf(nil, "$(file:%s)", filePath))
		require.NoError(t, err)
		require.Equal(t, "file content", string(result))
	})

	t.Run("url/http 200", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, "server response")
		}))
		defer server.Close()
		result, err := Substitute(t.Context(), fmt.Appendf(nil, "$(url:%s)", server.URL), WithHTTPClient(server.Client()))
		require.NoError(t, err)
		require.Equal(t, "server response", string(result))
	})

	t.Run("url/http non-200", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		_, err := Substitute(t.Context(), fmt.Appendf(nil, "$(url:%s)", server.URL), WithHTTPClient(server.Client()))
		require.ErrorContains(t, err, "HTTP status 404")
	})

	t.Run("url/file scheme", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		filePath := filepath.Join(dir, "remote.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("from file url"), 0600))
		// filePath is absolute (starts with /), so file:// + filePath gives file:///path/...
		result, err := Substitute(t.Context(), fmt.Appendf(nil, "$(url:file://%s)", filePath))
		require.NoError(t, err)
		require.Equal(t, "from file url", string(result))
	})

	t.Run("nested substitution", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "name.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("world"), 0600))
		t.Setenv("_TEST_NESTED_FILE_PATH", filePath)
		result, err := Substitute(t.Context(), []byte("$(file:$(env:_TEST_NESTED_FILE_PATH))"))
		require.NoError(t, err)
		require.Equal(t, "world", string(result))
	})

	t.Run("yaml-like config", func(t *testing.T) {
		t.Setenv("_TEST_YAML_BUCKET", "my-bucket")
		result, err := Substitute(t.Context(), []byte("buckets:\n  - name: $(env:_TEST_YAML_BUCKET)\n"))
		require.NoError(t, err)
		require.Equal(t, "buckets:\n  - name: my-bucket\n", string(result))
	})
}
