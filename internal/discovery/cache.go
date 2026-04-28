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
	cfg              *config.Config
	mu               sync.RWMutex
	targets          []TargetGroup
	lastErr          error
	prevTargetSet    map[string]struct{}
	failingTenancies map[string]bool
	cycleSeq         int64
}

// NewCache creates a Cache; call Start to begin background refresh.
func NewCache(cfg *config.Config) *Cache {
	return &Cache{
		cfg:              cfg,
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}
}

// Start performs an initial synchronous refresh (so the server starts with data)
// then launches a background goroutine that refreshes on the configured interval.
func (c *Cache) Start(ctx context.Context) {
	tagFilter := "disabled"
	if c.cfg.Discovery.TagKey != "" && c.cfg.Discovery.TagValue != "" {
		tagFilter = c.cfg.Discovery.TagKey + "=" + c.cfg.Discovery.TagValue
	}
	slog.Info("performing initial OCI discovery",
		"interval", c.cfg.Discovery.RefreshInterval,
		"tenancies", len(c.cfg.Tenancies),
		"tag_filter", tagFilter,
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

// refreshStats accumulates per-cycle metrics across all tenancies.
type refreshStats struct {
	tenanciesTotal      int
	tenanciesWithErrors int
	compartmentsFound   int
	compartmentsFailed  int
	errorTenancies      []string
}

// refresh discovers targets from all configured tenancies concurrently.
// A failure in one tenancy logs an error but does not prevent others from
// completing - the last successful full result is retained on partial failure.
func (c *Cache) refresh(ctx context.Context) {
	c.cycleSeq++
	cycleID := c.cycleSeq

	slog.Info("starting target refresh", "cycle_id", cycleID)
	start := time.Now()

	type result struct {
		groups []TargetGroup
		stats  tenancyStats
		err    error
		name   string
	}

	results := make(chan result, len(c.cfg.Tenancies))

	for _, tenancy := range c.cfg.Tenancies {
		go func(t config.TenancyConfig) {
			groups, stats, err := discoverTenancy(ctx, c.cfg, t)
			results <- result{groups: groups, stats: stats, err: err, name: t.Name}
		}(tenancy)
	}

	var (
		all      []TargetGroup
		anyError bool
	)

	rs := refreshStats{
		tenanciesTotal: len(c.cfg.Tenancies),
		errorTenancies: []string{},
	}

	// nowFailing tracks which tenancies degraded this cycle for transition detection.
	nowFailing := make(map[string]bool)

	for range c.cfg.Tenancies {
		r := <-results
		if r.err != nil {
			slog.Error("tenancy discovery failed",
				"cycle_id", cycleID,
				"tenancy", r.name,
				"error", r.err,
			)
			anyError = true
			rs.tenanciesWithErrors++
			rs.errorTenancies = append(rs.errorTenancies, r.name)
			nowFailing[r.name] = true
			continue
		}

		rs.compartmentsFound += r.stats.compartmentsDiscovered
		rs.compartmentsFailed += r.stats.compartmentsFailed

		if r.stats.hadErrors {
			anyError = true
			rs.tenanciesWithErrors++
			rs.errorTenancies = append(rs.errorTenancies, r.name)
			nowFailing[r.name] = true

			// Emit WARN only on transition: healthy -> degraded.
			// Persistent failures on every cycle produce no additional signal.
			if !c.failingTenancies[r.name] {
				slog.Warn("tenancy_discovery_complete",
					"cycle_id", cycleID,
					"tenancy", r.name,
					"compartments_ok", r.stats.compartmentsDiscovered,
					"compartments_failed", r.stats.compartmentsFailed,
					"error_code", r.stats.errorCode,
					"state", "degraded",
				)
			}
		} else if c.failingTenancies[r.name] {
			// Emit INFO on recovery: degraded -> healthy.
			slog.Info("tenancy_discovery_complete",
				"cycle_id", cycleID,
				"tenancy", r.name,
				"compartments_ok", r.stats.compartmentsDiscovered,
				"compartments_failed", 0,
				"state", "recovered",
			)
		}

		all = append(all, r.groups...)
	}

	// Update per-tenancy failure state for next cycle's transition check.
	c.failingTenancies = nowFailing

	var refreshErr error
	if anyError {
		refreshErr = fmt.Errorf("one or more tenancies failed during refresh")
	}

	// Compute targets delta only when the cache is actually updated.
	// If all tenancies failed we retain stale data, so the delta from Prometheus'
	// perspective is zero — logging targets_removed would be misleading.
	var added, removed, unchanged int
	if len(all) > 0 || !anyError {
		currentSet := targetSet(all)
		added, removed, unchanged = computeDelta(c.prevTargetSet, currentSet)
		c.mu.Lock()
		c.targets = all
		c.lastErr = refreshErr
		c.mu.Unlock()
		c.prevTargetSet = currentSet
	} else {
		unchanged = len(c.prevTargetSet)
	}

	// Use WARN when the cycle had any errors so alerting rules can match on level.
	logFn := slog.Info
	if anyError {
		logFn = slog.Warn
	}

	logFn("target_refresh_complete",
		"cycle_id", cycleID,
		"duration_ms", time.Since(start).Milliseconds(),
		"total_groups", len(all),
		"had_errors", anyError,
		"tenancies_total", rs.tenanciesTotal,
		"tenancies_with_errors", rs.tenanciesWithErrors,
		"compartments_discovered", rs.compartmentsFound,
		"compartments_failed", rs.compartmentsFailed,
		"error_tenancies", rs.errorTenancies,
		"targets_added", added,
		"targets_removed", removed,
		"targets_unchanged", unchanged,
	)
}

// targetSet builds a flat set of all target addresses across all groups.
func targetSet(groups []TargetGroup) map[string]struct{} {
	s := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		for _, t := range g.Targets {
			s[t] = struct{}{}
		}
	}
	return s
}

// computeDelta returns counts of added, removed, and unchanged targets
// between the previous and current target sets.
func computeDelta(prev, curr map[string]struct{}) (added, removed, unchanged int) {
	for t := range curr {
		if _, ok := prev[t]; ok {
			unchanged++
		} else {
			added++
		}
	}
	for t := range prev {
		if _, ok := curr[t]; !ok {
			removed++
		}
	}
	return
}
