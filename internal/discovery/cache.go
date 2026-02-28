package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
)

// Cache holds the most recently discovered targets and refreshes them
// in the background at the configured interval. HTTP handlers call Get()
// which always returns instantly from memory.
type Cache struct {
	cfg     *config.Config
	mu      sync.RWMutex
	targets []TargetGroup
	lastErr error
}

// NewCache creates a Cache; call Start to begin background refresh.
func NewCache(cfg *config.Config) *Cache {
	return &Cache{cfg: cfg}
}

// Start performs an initial synchronous refresh (so the server starts with data)
// then launches a background goroutine that refreshes on the configured interval.
func (c *Cache) Start(ctx context.Context) {
	slog.Info("performing initial OCI discovery",
		"interval", c.cfg.Discovery.RefreshInterval,
		"tenancies", len(c.cfg.Tenancies),
	)
	c.refresh(ctx)

	go func() {
		ticker := time.NewTicker(c.cfg.Discovery.RefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.refresh(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Get returns the current cached target list. Returns an empty slice if no
// targets have been discovered yet.
func (c *Cache) Get() []TargetGroup {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.targets == nil {
		return []TargetGroup{}
	}
	return c.targets
}

// LastError returns the error from the most recent refresh attempt, if any.
func (c *Cache) LastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastErr
}

// refresh discovers targets from all configured tenancies concurrently.
// A failure in one tenancy logs an error but does not prevent others from
// completing - the last successful full result is retained on partial failure.
func (c *Cache) refresh(ctx context.Context) {
	slog.Info("starting target refresh")
	start := time.Now()

	type result struct {
		groups []TargetGroup
		err    error
		name   string
	}

	results := make(chan result, len(c.cfg.Tenancies))

	for _, tenancy := range c.cfg.Tenancies {
		go func(t config.TenancyConfig) {
			groups, err := discoverTenancy(ctx, c.cfg, t)
			results <- result{groups: groups, err: err, name: t.Name}
		}(tenancy)
	}

	var (
		all      []TargetGroup
		anyError bool
	)

	for range c.cfg.Tenancies {
		r := <-results
		if r.err != nil {
			slog.Error("tenancy discovery failed",
				"tenancy", r.name,
				"error", r.err,
			)
			anyError = true
			continue
		}
		slog.Info("tenancy discovery complete",
			"tenancy", r.name,
			"target_groups", len(r.groups),
		)
		all = append(all, r.groups...)
	}

	var refreshErr error
	if anyError {
		refreshErr = fmt.Errorf("one or more tenancies failed during refresh")
	}

	// Only update the cache when at least some results came back
	if len(all) > 0 || !anyError {
		c.mu.Lock()
		c.targets = all
		c.lastErr = refreshErr
		c.mu.Unlock()
	}

	slog.Info("target refresh complete",
		"total_groups", len(all),
		"duration_ms", time.Since(start).Milliseconds(),
		"had_errors", anyError,
	)
}
