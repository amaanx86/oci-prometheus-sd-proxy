package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/discovery"
)

// mockCache implements TargetCache for testing.
type mockCache struct {
	targets []discovery.TargetGroup
}

func (m *mockCache) Get() []discovery.TargetGroup {
	return m.targets
}

func TestTargets_GET_ReturnsOK(t *testing.T) {
	cache := &mockCache{
		targets: []discovery.TargetGroup{
			{
				Targets: []string{"10.0.0.1:9100"},
				Labels:  map[string]string{"__meta_oci_region": "us-ashburn-1"},
			},
		},
	}

	handler := Targets(cache)
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestTargets_ContentTypeJSON(t *testing.T) {
	handler := Targets(&mockCache{targets: []discovery.TargetGroup{}})
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json; charset=utf-8")
	}
}

func TestTargets_EmptyCacheReturnsEmptyArray(t *testing.T) {
	handler := Targets(&mockCache{targets: []discovery.TargetGroup{}})
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}

	var result []discovery.TargetGroup
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestTargets_NonGETMethodNotAllowed(t *testing.T) {
	handler := Targets(&mockCache{})

	methods := []string{
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/v1/targets", nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s: got %d, want 405", method, rr.Code)
			}
		})
	}
}

func TestTargets_ResponseBodyIsValidJSON(t *testing.T) {
	cache := &mockCache{
		targets: []discovery.TargetGroup{
			{
				Targets: []string{"10.0.0.1:9100", "10.0.0.2:9100"},
				Labels: map[string]string{
					"__meta_oci_region":        "us-ashburn-1",
					"__meta_oci_tenancy_name":  "prod",
					"__meta_oci_instance_name": "my-instance",
				},
			},
			{
				Targets: []string{"10.0.0.3:9182"},
				Labels:  map[string]string{"__meta_oci_region": "eu-frankfurt-1"},
			},
		},
	}

	handler := Targets(cache)
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var result []discovery.TargetGroup
	if err := json.NewDecoder(rr.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("result count: got %d, want 2", len(result))
	}

	if len(result[0].Targets) != 2 {
		t.Errorf("first group targets: got %d, want 2", len(result[0].Targets))
	}
	if result[0].Labels["__meta_oci_region"] != "us-ashburn-1" {
		t.Errorf("label: got %q, want %q", result[0].Labels["__meta_oci_region"], "us-ashburn-1")
	}
}

func TestTargets_Always200EvenWithData(t *testing.T) {
	// Prometheus HTTP SD contract: always return 200, never 204 or 5xx
	cache := &mockCache{
		targets: []discovery.TargetGroup{
			{Targets: []string{"1.2.3.4:9100"}, Labels: map[string]string{}},
		},
	}
	handler := Targets(cache)
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestTargets_JSONContainsTargetsAndLabelsKeys(t *testing.T) {
	cache := &mockCache{
		targets: []discovery.TargetGroup{
			{
				Targets: []string{"10.0.0.1:9100"},
				Labels:  map[string]string{"job": "node"},
			},
		},
	}
	handler := Targets(cache)
	req := httptest.NewRequest(http.MethodGet, "/v1/targets", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Decode as a generic slice to check key presence
	var raw []map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected at least one group")
	}
	if _, ok := raw[0]["targets"]; !ok {
		t.Error("response missing 'targets' key")
	}
	if _, ok := raw[0]["labels"]; !ok {
		t.Error("response missing 'labels' key")
	}
}
