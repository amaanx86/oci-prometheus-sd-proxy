# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-03-21

### Added

- **Canonical refresh log line**: Single `target_refresh_complete` log line per cycle with `cycle_id`, `duration_ms`, `total_groups`, `had_errors`, `tenancies_total`, `tenancies_with_errors`, `compartments_discovered`, `compartments_failed`, `error_tenancies`, `targets_added`, `targets_removed`, `targets_unchanged`
- **Targets delta tracking**: Each refresh cycle reports how many targets were added, removed, or unchanged compared to the previous cycle, making silent target loss visible
- **Cycle ID**: Monotonic `cycle_id` field on all refresh-related log lines for correlation across sub-logs
- **Tenancy state transition logging**: `tenancy_discovery_complete` WARN fires only on healthy-to-degraded transition; INFO fires on recovery - persistent failures produce no repeated noise

### Changed

- `target_refresh_complete` emits at WARN level when `had_errors=true`, INFO otherwise - enables log-level-based alerting without custom field matchers
- Per-compartment `failed to list child compartments` log demoted from WARN to DEBUG - eliminates ~11,500 redundant log lines/day for persistent IAM failures
- `refreshStats` struct introduced to accumulate per-cycle metrics across all tenancies

### Fixed

- `had_errors=false` on `target_refresh_complete` when compartment-level failures occurred in the same cycle - propagation now flows correctly from compartment stats through tenancy stats to the cycle summary
- `targets_removed` falsely reported when all tenancies fail and stale cache is retained - delta now reflects what was actually committed to cache, not the theoretical diff
- Network and TLS errors now classified as `error_code: network_error` instead of `unknown` - `*url.Error` (wraps TLS, DNS, HTTP transport failures) and `*net.OpError` are both detected; context cancellations and deadlines classified as `context_canceled` and `timeout` respectively
- OCI service error codes (e.g. `NotAuthorizedOrNotFound`) now correctly extracted when wrapped - `extractErrorCode` walks the full error chain instead of using a direct type assertion, fixing cases where the SDK wraps the underlying `servicefailure` type

## [1.1.0] - 2026-03-10

### Added

- **OCI API Rate Limiting**: Implement proactive token bucket rate limiting to prevent 429 TooManyRequests errors
  - Default rate: 10 requests/second per tenancy
  - Configurable via `DISCOVERY_RATE_LIMIT_RPS` environment variable or `discovery.rate_limit_rps` config field
  - Prevents compartment discovery failures due to API throttling

- **OCI SDK Retry Policy**: Automatically retry transient failures with exponential backoff
  - Applied to all OCI API calls (ListInstances, ListCompartments, ListVnicAttachments, GetVnic)
  - Handles 429 status codes with up to 8 retry attempts and max 30-second sleep
  - Ensures discovered compartments are never permanently skipped due to transient errors

- **Rate Limiter Configuration**: New `rate_limit_rps` field in discovery config
  - Type: float64
  - Default: 10.0 requests per second
  - Environment variable: `DISCOVERY_RATE_LIMIT_RPS`
  - Burst capacity: equal to rate (at least 1)

- **Struct Refactoring**: Introduce `tenancyDiscoverer` struct for cleaner state management
  - Centralizes rate limiter, retry policy, and OCI clients in single struct
  - Methods: `discover()`, `discoverCompartment()`, `listAllInstances()`, `listAllCompartments()`, `getPrimaryPrivateIP()`
  - Rate limiting applied before every API call via `limiter.Wait(ctx)`

### Changed

- **Dependency Update**: Added `golang.org/x/time v0.5.0` for token bucket implementation

### Fixed

- Compartments no longer skip permanently when hitting OCI API 429 rate limits
- Transient API failures are now retried automatically instead of failing discovery

### Technical Details

- **Belt & Suspenders Approach**: Two-layer defense against rate limiting
  1. Proactive: Token bucket limits outgoing requests before they hit the API
  2. Reactive: OCI SDK DefaultRetryPolicy handles 429 responses with backoff
- **Per-Tenancy Limiting**: Each tenancy runs in its own goroutine with its own rate limiter
- **Zero API Changes**: Public API and cache interface remain unchanged

## [1.0.0] - 2026-03-02

### Added

- **Multi-Tenancy Discovery**: Support for discovering compute instances across multiple OCI tenancies
  - Parallel tenancy scanning with configurable refresh interval
  - Auto-discovery of all compartments or explicit compartment list
  - Automatic fallback to root compartment on discovery failure

- **Tag-Based Filtering**: Filter instances by freeform or defined tags
  - Configurable tag key and value for instance selection
  - Support for instance discovery based on monitoring tags

- **Prometheus HTTP Service Discovery**: Full implementation of Prometheus HTTP SD API
  - Endpoint: `GET /v1/targets`
  - Returns targets with rich metadata for relabeling
  - Bearer token authentication

- **Rich Metadata Labels**: OCI-specific labels for Prometheus relabeling
  - Tenancy name, ID, and region
  - Compartment ID
  - Instance ID, name, state, and shape
  - Availability domain and fault domain
  - Image ID and private IP
  - All freeform instance tags

- **In-Memory Caching**: Fast target group caching with background refresh
  - Configurable refresh interval (default 5 minutes)
  - Partial results on compartment discovery errors
  - Background goroutine keeps cache fresh

- **Configuration Management**: YAML-based config with environment variable overrides
  - Server: port, bearer token
  - Discovery: tag filters, port overrides, refresh interval
  - Tenancies: credentials, regions, compartment scope
  - All scalar values can be overridden via environment variables

- **Multiple OCI Clients**: Support for compute, network, and identity API clients
  - List instances, compartments, VNICs
  - Resolve primary private IPs
  - Handle instance metadata and relationships

- **Health Checks**: Liveness and readiness probes
  - `/healthz` - Liveness probe
  - `/readyz` - Readiness probe with cache status

- **Security Features**
  - Bearer token authentication on all endpoints
  - Distroless container image for minimal attack surface
  - Read-only config and key mounts in Docker
  - Private key support with optional passphrase

- **Production-Ready Observability**
  - Structured JSON logging with timestamps and levels
  - Request logging middleware
  - Configurable logging level
  - Clear error messages for debugging

- **Docker Support**
  - Multi-stage Dockerfile with optimized layers
  - docker-compose configuration for local development
  - Docker image support for production deployment

- **Development Tools**
  - Makefile with run, test, build, and clean targets
  - GitHub Actions CI/CD for testing and Docker image building
  - CodeQL security scanning

[Unreleased]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/releases/tag/v1.0.0
