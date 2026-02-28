// Package server wires up the HTTP mux, middleware chain, and server instance.
package server

import (
	"fmt"
	"net/http"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/discovery"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/handler"
	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/middleware"
)

// New creates and returns a configured *http.Server.
// The caller is responsible for calling ListenAndServe and Shutdown.
func New(cfg *config.Config, cache *discovery.Cache) *http.Server {
	mux := http.NewServeMux()

	// /v1/targets - Prometheus HTTP SD endpoint (requires Bearer token)
	targetsHandler := middleware.Logging(
		middleware.BearerAuth(cfg.Server.Token,
			handler.Targets(cache),
		),
	)
	mux.Handle("/v1/targets", targetsHandler)

	// /healthz - unauthenticated liveness probe for load balancers / k8s
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// /readyz - readiness probe (same as healthz for now; could check cache age)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}
}
