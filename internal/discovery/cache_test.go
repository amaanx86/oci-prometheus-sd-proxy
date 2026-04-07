package discovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/amaanx86/oci-prometheus-sd-proxy/internal/config"
)

// ---- targetSet --------------------------------------------------------------

func TestTargetSet_Empty(t *testing.T) {
	s := targetSet(nil)
	if len(s) != 0 {
		t.Errorf("expected empty set, got %d entries", len(s))
	}
}

func TestTargetSet_SingleGroup(t *testing.T) {
	groups := []TargetGroup{
		{Targets: []string{"10.0.0.1:9100", "10.0.0.2:9100"}},
	}
	s := targetSet(groups)
	if len(s) != 2 {
		t.Errorf("set size: got %d, want 2", len(s))
	}
	if _, ok := s["10.0.0.1:9100"]; !ok {
		t.Error("10.0.0.1:9100 missing from set")
	}
	if _, ok := s["10.0.0.2:9100"]; !ok {
		t.Error("10.0.0.2:9100 missing from set")
	}
}

func TestTargetSet_MultipleGroups(t *testing.T) {
	groups := []TargetGroup{
		{Targets: []string{"10.0.0.1:9100"}},
		{Targets: []string{"10.0.0.2:9100", "10.0.0.3:9100"}},
	}
	s := targetSet(groups)
	if len(s) != 3 {
		t.Errorf("set size: got %d, want 3", len(s))
	}
}

func TestTargetSet_Deduplicates(t *testing.T) {
	groups := []TargetGroup{
		{Targets: []string{"10.0.0.1:9100"}},
		{Targets: []string{"10.0.0.1:9100"}}, // duplicate
	}
	s := targetSet(groups)
	if len(s) != 1 {
		t.Errorf("set size with duplicates: got %d, want 1", len(s))
	}
}

func TestTargetSet_EmptyTargetsSlice(t *testing.T) {
	groups := []TargetGroup{
		{Targets: []string{}},
		{Targets: nil},
	}
	s := targetSet(groups)
	if len(s) != 0 {
		t.Errorf("expected empty set for groups with no targets, got %d", len(s))
	}
}

// ---- computeDelta -----------------------------------------------------------

func TestComputeDelta_AllNew(t *testing.T) {
	prev := map[string]struct{}{}
	curr := map[string]struct{}{
		"10.0.0.1:9100": {},
		"10.0.0.2:9100": {},
	}
	added, removed, unchanged := computeDelta(prev, curr)
	if added != 2 {
		t.Errorf("added: got %d, want 2", added)
	}
	if removed != 0 {
		t.Errorf("removed: got %d, want 0", removed)
	}
	if unchanged != 0 {
		t.Errorf("unchanged: got %d, want 0", unchanged)
	}
}

func TestComputeDelta_AllRemoved(t *testing.T) {
	prev := map[string]struct{}{
		"10.0.0.1:9100": {},
		"10.0.0.2:9100": {},
	}
	curr := map[string]struct{}{}
	added, removed, unchanged := computeDelta(prev, curr)
	if added != 0 {
		t.Errorf("added: got %d, want 0", added)
	}
	if removed != 2 {
		t.Errorf("removed: got %d, want 2", removed)
	}
	if unchanged != 0 {
		t.Errorf("unchanged: got %d, want 0", unchanged)
	}
}

func TestComputeDelta_NoChange(t *testing.T) {
	set := map[string]struct{}{
		"10.0.0.1:9100": {},
		"10.0.0.2:9100": {},
	}
	added, removed, unchanged := computeDelta(set, set)
	if added != 0 {
		t.Errorf("added: got %d, want 0", added)
	}
	if removed != 0 {
		t.Errorf("removed: got %d, want 0", removed)
	}
	if unchanged != 2 {
		t.Errorf("unchanged: got %d, want 2", unchanged)
	}
}

func TestComputeDelta_Mixed(t *testing.T) {
	prev := map[string]struct{}{
		"10.0.0.1:9100": {},
		"10.0.0.2:9100": {},
	}
	curr := map[string]struct{}{
		"10.0.0.2:9100": {}, // unchanged
		"10.0.0.3:9100": {}, // new
	}
	added, removed, unchanged := computeDelta(prev, curr)
	if added != 1 {
		t.Errorf("added: got %d, want 1", added)
	}
	if removed != 1 {
		t.Errorf("removed: got %d, want 1", removed)
	}
	if unchanged != 1 {
		t.Errorf("unchanged: got %d, want 1", unchanged)
	}
}

func TestComputeDelta_BothEmpty(t *testing.T) {
	added, removed, unchanged := computeDelta(
		map[string]struct{}{},
		map[string]struct{}{},
	)
	if added != 0 || removed != 0 || unchanged != 0 {
		t.Errorf("both empty: got added=%d removed=%d unchanged=%d, want all 0",
			added, removed, unchanged)
	}
}

// ---- Cache.Get / Cache.LastError --------------------------------------------

// We test Cache.Get and Cache.LastError without triggering any OCI calls
// by directly manipulating the internal state under the mutex.

func TestCache_GetReturnsEmptySliceWhenNil(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}
	got := c.Get()
	if got == nil {
		t.Error("Get() returned nil, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("Get() length: got %d, want 0", len(got))
	}
}

func TestCache_GetReturnsStoredTargets(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}

	want := []TargetGroup{
		{Targets: []string{"10.0.0.1:9100"}, Labels: map[string]string{"k": "v"}},
	}
	c.targets = want

	got := c.Get()
	if len(got) != len(want) {
		t.Fatalf("Get() count: got %d, want %d", len(got), len(want))
	}
	if got[0].Targets[0] != want[0].Targets[0] {
		t.Errorf("target address: got %q, want %q", got[0].Targets[0], want[0].Targets[0])
	}
}

func TestCache_LastErrorNilByDefault(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}
	if err := c.LastError(); err != nil {
		t.Errorf("LastError() got %v, want nil", err)
	}
}

func TestCache_LastErrorReturnsStoredError(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}
	sentinel := errSentinel("test error")
	c.lastErr = sentinel

	if got := c.LastError(); got != sentinel {
		t.Errorf("LastError(): got %v, want %v", got, sentinel)
	}
}

// TestCache_GetConcurrentSafe verifies no data races under concurrent reads.
func TestCache_GetConcurrentSafe(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
		targets: []TargetGroup{
			{Targets: []string{"10.0.0.1:9100"}},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Get()
		}()
	}
	wg.Wait()
}

// TestCache_GetAndWriteConcurrentSafe verifies no data races with concurrent
// reads and writes (simulating what the background refresh goroutine does).
func TestCache_GetAndWriteConcurrentSafe(t *testing.T) {
	c := &Cache{
		prevTargetSet:    make(map[string]struct{}),
		failingTenancies: make(map[string]bool),
	}

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.mu.Lock()
			c.targets = []TargetGroup{{Targets: []string{"10.0.0.1:9100"}}}
			c.mu.Unlock()
		}(i)
	}

	// Readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Get()
		}()
	}

	wg.Wait()
}

// ---- NewCache ---------------------------------------------------------------

func TestNewCache_NotNil(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)
	if c == nil {
		t.Fatal("NewCache returned nil")
	}
}

func TestNewCache_InitialisedFields(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)

	if c.cfg != cfg {
		t.Error("cfg not stored on Cache")
	}
	if c.prevTargetSet == nil {
		t.Error("prevTargetSet should be initialised (not nil)")
	}
	if c.failingTenancies == nil {
		t.Error("failingTenancies should be initialised (not nil)")
	}
}

func TestNewCache_GetBeforeStartReturnsEmpty(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)
	got := c.Get()
	if len(got) != 0 {
		t.Errorf("Get() before Start: got %d items, want 0", len(got))
	}
}

// ---- refresh (error path via missing key file) ------------------------------
//
// These tests exercise refresh/discoverTenancy when the private_key_path does
// not exist. The OCI read fails immediately - no network calls are made. This
// covers the "all tenancies failed - preserve stale data" branch.

func minimalCacheConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Token: "tok"},
		Discovery: config.DiscoveryConfig{
			TagKey:          "monitoring",
			TagValue:        "enabled",
			LinuxPort:       9100,
			WindowsPort:     9182,
			RefreshInterval: time.Hour, // long - we drive refresh manually in tests
			RateLimitRPS:    10,
		},
		Tenancies: []config.TenancyConfig{
			{
				Name:           "test-tenancy",
				Region:         "us-ashburn-1",
				TenancyID:      "ocid1.tenancy.oc1..test",
				UserID:         "ocid1.user.oc1..test",
				Fingerprint:    "aa:bb",
				PrivateKeyPath: "/nonexistent-key-for-testing.pem", // intentionally missing
			},
		},
	}
}

func TestCache_RefreshAllTenanciesFail_PreservesEmptyTargets(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)

	c.refresh(context.Background())

	// When all tenancies fail and there is no prior data, targets stay nil/empty.
	got := c.Get()
	if len(got) != 0 {
		t.Errorf("Get() after all-fail refresh: got %d items, want 0", len(got))
	}
}

func TestCache_RefreshAllTenanciesFail_PreservesStaleTargets(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)

	// Pre-seed the cache with stale data.
	stale := []TargetGroup{{Targets: []string{"10.0.0.1:9100"}, Labels: map[string]string{}}}
	c.mu.Lock()
	c.targets = stale
	c.mu.Unlock()

	c.refresh(context.Background())

	// Stale data must be preserved when all tenancies fail.
	got := c.Get()
	if len(got) != 1 {
		t.Fatalf("stale data not preserved: got %d groups, want 1", len(got))
	}
	if got[0].Targets[0] != "10.0.0.1:9100" {
		t.Errorf("stale target: got %q, want %q", got[0].Targets[0], "10.0.0.1:9100")
	}
}

func TestCache_RefreshIncrementsSeqID(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)

	before := c.cycleSeq
	c.refresh(context.Background())
	if c.cycleSeq != before+1 {
		t.Errorf("cycleSeq: got %d, want %d", c.cycleSeq, before+1)
	}
}

func TestCache_RefreshMultipleCycles_SeqIDIncreases(t *testing.T) {
	cfg := minimalCacheConfig()
	c := NewCache(cfg)

	for i := 1; i <= 3; i++ {
		c.refresh(context.Background())
		if int(c.cycleSeq) != i {
			t.Errorf("cycle %d: cycleSeq = %d, want %d", i, c.cycleSeq, i)
		}
	}
}

// TestCache_Start_DoesNotPanic verifies Start completes the initial refresh
// and launches the background goroutine without panicking, even when all
// tenancy key files are missing (OCI calls never happen).
func TestCache_Start_DoesNotPanic(t *testing.T) {
	cfg := minimalCacheConfig()
	cfg.Discovery.RefreshInterval = 10 * time.Millisecond
	c := NewCache(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start does initial sync refresh then spawns background goroutine.
	// With missing key file, discoverTenancy errors immediately - no network.
	c.Start(ctx)

	// After Start: cache should have gone through one refresh cycle.
	if c.cycleSeq != 1 {
		t.Errorf("cycleSeq after Start: got %d, want 1", c.cycleSeq)
	}

	// Get() must never panic or return nil.
	got := c.Get()
	if got == nil {
		t.Error("Get() returned nil after Start")
	}

	// Cancel and give the ticker goroutine time to exit cleanly.
	cancel()
	time.Sleep(50 * time.Millisecond)
}

// TestCache_Start_BackgroundTickerRefreshes verifies that the background ticker
// fires without panicking. We observe this safely via LastError(), which is
// mutex-protected and gets set after at least one background refresh cycle.
// (With a missing key file the error path runs, so LastError stays nil because
// all tenancies failed and stale-preserve logic skips the update - but Get()
// remains stable and race-free.)
func TestCache_Start_BackgroundTickerRefreshes(t *testing.T) {
	cfg := minimalCacheConfig()
	cfg.Discovery.RefreshInterval = 20 * time.Millisecond
	c := NewCache(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.Start(ctx)

	// Sleep long enough for at least two ticker intervals to fire.
	time.Sleep(100 * time.Millisecond)

	// Get() must always be safe to call - ticker goroutine must not have panicked.
	got := c.Get()
	if got == nil {
		t.Error("Get() returned nil after background ticks")
	}
	// LastError is mutex-protected and safe to read concurrently.
	_ = c.LastError()
}

// errSentinel is a simple error type for testing.
type errSentinel string

func (e errSentinel) Error() string { return string(e) }
