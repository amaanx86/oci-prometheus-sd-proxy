// Package handler implements HTTP handlers for the Prometheus HTTP SD API.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/discovery"
)

// TargetCache is satisfied by *discovery.Cache.
type TargetCache interface {
	Get() []discovery.TargetGroup
}

// Targets returns an http.HandlerFunc that serves the Prometheus HTTP SD response.
// It reads from the in-memory cache and responds with Content-Type: application/json.
//
// Prometheus HTTP SD contract:
//   - GET only
//   - HTTP 200 always (even when list is empty)
//   - Content-Type: application/json; charset=utf-8
//   - Full target list on every request (no incremental updates)
func Targets(cache TargetCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		targets := cache.Get()

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(targets)
	}
}
