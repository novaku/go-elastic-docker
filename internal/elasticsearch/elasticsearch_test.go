package elasticsearch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/novaku/go-elastic-search/config"
	"go.uber.org/zap"
)

func TestBuildTransport_WithAndWithoutCACert(t *testing.T) {
	t.Run("without cert", func(t *testing.T) {
		tr := buildTransport(&config.ESConfig{})
		if tr.TLSClientConfig == nil {
			t.Fatalf("expected TLS config")
		}
		if tr.TLSClientConfig.RootCAs != nil {
			t.Fatalf("expected nil RootCAs when cert missing")
		}
	})

	t.Run("with cert", func(t *testing.T) {
		d := t.TempDir()
		certPath := filepath.Join(d, "ca.pem")
		pem := "-----BEGIN CERTIFICATE-----\nMIIBhTCCASugAwIBAgIRALa3u7Vv3kSLm0wM2w8rjX0wCgYIKoZIzj0EAwIwEzER\nMA8GA1UEAwwIdGVzdC1jYTAeFw0yNDAxMDEwMDAwMDBaFw0zNDAxMDEwMDAwMDBa\nMBMxETAPBgNVBAMMCHRlc3QtY2EwWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQW\n1fPBXx4dTH+8wXfYh2UbdQzShk39lN7XKx0z2QxLxmI2d7Ewz0hG84w6M+by83mM\n8f7c8j4ocVvZr0jtv7Eoo0UwQzAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUw\nAwEB/zAdBgNVHQ4EFgQU4V2n9YxvZx6vQjWmT8Yh5x3mM6AwCgYIKoZIzj0EAwID\nRwAwRAIgXyW6kC8hQ2OaK7vVqB6+XHf6G6M8lM7l8sYQ2CZ4VQ4CIH6aR5uA0jE7\nLw3qYI0w8u9gJQ+8r8Xk3o9gV0y9n3sG\n-----END CERTIFICATE-----\n"
		if err := os.WriteFile(certPath, []byte(pem), 0o600); err != nil {
			t.Fatalf("write cert: %v", err)
		}

		tr := buildTransport(&config.ESConfig{CACert: certPath})
		if tr.TLSClientConfig == nil || tr.TLSClientConfig.RootCAs == nil {
			t.Fatalf("expected RootCAs to be set")
		}
	})
}

func TestClient_NewAndPing(t *testing.T) {
	t.Run("new success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			if r.Method == http.MethodHead && r.URL.Path == "/" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "{}")
		}))
		defer ts.Close()

		cfg := &config.ESConfig{Addresses: []string{ts.URL}}
		c, err := New(cfg, zap.NewNop())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if c == nil || c.ES == nil {
			t.Fatalf("expected client")
		}
	})

	t.Run("new ping failure", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Elastic-Product", "Elasticsearch")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, "down")
		}))
		defer ts.Close()

		cfg := &config.ESConfig{Addresses: []string{ts.URL}}
		_, err := New(cfg, zap.NewNop())
		if err == nil {
			t.Fatalf("expected error")
		}
	})
}
