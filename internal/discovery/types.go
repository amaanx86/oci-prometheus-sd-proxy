// Package discovery implements OCI instance discovery and target caching
// for the Prometheus HTTP Service Discovery API.
package discovery

// TargetGroup represents a single Prometheus HTTP SD target group.
// Each group contains one or more targets that share the same label set.
type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}
