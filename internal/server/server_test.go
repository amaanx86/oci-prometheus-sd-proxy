package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/discovery"
)

// testConfig returns a minimal valid config for tests.
func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:  8080,
			Token: "test-token",
		},
		Discovery: config.DiscoveryConfig{
			TagKey:      "monitoring",
			TagValue:    "enabled",
			LinuxPort:   9100,
			WindowsPort: 9182,
		},
		Tenancies: []config.TenancyConfig{
			{
				Name:           "test",
				Region:         "us-ashburn-1",
				TenancyID:      "ocid1.tenancy.oc1..test",
				UserID:         "ocid1.user.oc1..test",
				Fingerprint:    "aa:bb:cc",
				PrivateKeyPath: "/tmp/key.pem",
			},
		},
	}
}

// ---- helper: spin up a test server from New() -------------------------------

func newTestServer(t *testing.T) (*httptest.Server, *config.Config, *discovery.Cache) {
	t.Helper()
	cfg := testConfig()
	cache := discovery.NewCache(cfg)
	srv := New(cfg, cache)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts, cfg, cache
}

// ---- /healthz ---------------------------------------------------------------

func TestHealthz_Returns200(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestHealthz_ReturnsOKBody(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok\n" {
		t.Errorf("body: got %q, want %q", string(body), "ok\n")
	}
}

func TestHealthz_ContentTypePlainText(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "text/plain" {
		t.Errorf("Content-Type: got %q, want %q", ct, "text/plain")
	}
}

func TestHealthz_NoAuthRequired(t *testing.T) {
	ts, _, _ := newTestServer(t)
	// No Authorization header - should still get 200
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200 (no auth required for healthz)", resp.StatusCode)
	}
}

// ---- /readyz ----------------------------------------------------------------

func TestReadyz_Returns200(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestReadyz_ReturnsOKBody(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok\n" {
		t.Errorf("body: got %q, want %q", string(body), "ok\n")
	}
}

func TestReadyz_NoAuthRequired(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200 (no auth required for readyz)", resp.StatusCode)
	}
}

// ---- /v1/targets authentication ---------------------------------------------

func TestTargetsRoute_Unauthenticated(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/v1/targets")
	if err != nil {
		t.Fatalf("GET /v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", resp.StatusCode)
	}
}

func TestTargetsRoute_WrongToken(t *testing.T) {
	ts, _, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/targets", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", resp.StatusCode)
	}
}

func TestTargetsRoute_ValidToken_Returns200(t *testing.T) {
	ts, cfg, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/targets", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Server.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}

func TestTargetsRoute_ValidToken_ContentTypeJSON(t *testing.T) {
	ts, cfg, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/v1/targets", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Server.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/targets: %v", err)
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json; charset=utf-8")
	}
}

func TestTargetsRoute_POST_MethodNotAllowed(t *testing.T) {
	ts, cfg, _ := newTestServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/targets", nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Server.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /v1/targets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", resp.StatusCode)
	}
}

// ---- server address ---------------------------------------------------------

func TestNew_ServerAddr(t *testing.T) {
	cfg := testConfig()
	cfg.Server.Port = 9999
	cache := discovery.NewCache(cfg)
	srv := New(cfg, cache)

	if srv.Addr != ":9999" {
		t.Errorf("Addr: got %q, want %q", srv.Addr, ":9999")
	}
}

// ---- unknown routes ---------------------------------------------------------

func TestUnknownRoute_Returns404(t *testing.T) {
	ts, _, _ := newTestServer(t)
	resp, err := http.Get(ts.URL + "/does-not-exist")
	if err != nil {
		t.Fatalf("GET /does-not-exist: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want 404", resp.StatusCode)
	}
}
