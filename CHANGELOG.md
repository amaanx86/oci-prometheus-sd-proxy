# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.1] - 2026-04-21

### Changed

- Bumped `github.com/oracle/oci-go-sdk/v65` from `65.111.0` to `65.112.0`

## [1.5.0] - 2026-04-18

### Added

- Instance principal authentication support for OCI tenancies. Set `auth_type: instance_principal` on a tenancy to authenticate via OCI IMDS instead of static user/key credentials. No `user_id`, `fingerprint`, or `private_key_path` fields required. Useful when the proxy runs directly on an OCI compute instance with a dynamic group + IAM policy granting read access.
- `auth_type` field on tenancy config accepting `api_key` (default, existing behaviour unchanged) or `instance_principal`. Unknown values are rejected at startup with a clear error.

### Tests

- Added `TestValidate_InstancePrincipalAuth`: asserts instance principal tenancies pass validation without credential fields.
- Added `TestValidate_InstancePrincipalStillRequiresRegionAndTenancyID`: asserts `name`, `tenancy_id`, and `region` remain required for instance principal tenancies.
- Added `TestValidate_UnknownAuthType`: asserts an unrecognised `auth_type` value is rejected at startup.

### Documentation

- `docs/installation.rst`: added Option B section with step-by-step dynamic group creation, instance principal IAM policy statements, and a note that policies must be at tenancy root level for compartment auto-discovery.
- `docs/configuration.rst`: documented `auth_type` field and clarified which credential fields are conditional on auth method.
- `README.md`: updated OCI IAM Policy section to show both API key and instance principal policy blocks.

## [1.4.2] - 2026-04-12

### Fixed

- TUF release manifests were not being registered as TUF targets - `metadata/targets.json` was never updated by the release workflow, so `targets.json` always had an empty target list and a TUF client could not verify any release manifest through the TUF chain
- Signing event PRs became mergeable immediately without offline maintainer signing because no metadata change was present on the signing branch, so the existing signature on `targets.json` satisfied the threshold check without any new signing required
- `cosign verify` and `cosign verify-attestation` commands in release verification manifests were missing `--certificate-identity` and `--certificate-oidc-issuer` flags, meaning verification would pass for any Sigstore signer rather than being pinned to the release workflow identity
- Redeclared `ARG VERSION` in the final Dockerfile stage to resolve undefined variable warning on `LABEL org.opencontainers.image.version`

### CI

- Release workflow now uses `python-tuf` to add the release manifest SHA-256 hash and length to `metadata/targets.json` and clear its signatures before pushing to the signing branch, ensuring the signing-event action requires a fresh offline signature from the maintainer before the PR can be merged
- Bumped `actions/checkout` from `v4` to `v6.0.2` across all workflows
- Bumped `docker/setup-buildx-action` from `v3` to `v4.0.0`
- Bumped `docker/login-action` from `v4` to `v4.1.0`
- Bumped `docker/metadata-action` from `v6` to `v6.0.0`
- Bumped `docker/build-push-action` from `v7` to `v7.1.0`
- Bumped `actions/setup-go` from `v6` to `v6.4.0`
- Pinned `aquasecurity/trivy-action` from `@master` to `v0.35.0`

## [1.4.1] - 2026-04-11

### Security

- Sign release images with cosign using GitHub Actions OIDC (keyless signing via Sigstore) on every release
- Attach a `release-metadata.json` attestation to each image recording version, digest, commit, and workflow run URL - verifiable with `cosign verify-attestation`
- Upload `release-metadata.json` as a GitHub release asset for out-of-band verification
- Publish a signed release verification manifest to the TUF-on-CI repository (`amaanx86/oci-prometheus-sd-proxy-tuf-on-ci`) on each release, enabling TUF-based trust anchoring for release metadata
- Added SLSA Level 3 provenance attestation for release container images via `slsa-framework/slsa-github-generator`
- Added CODEOWNERS file requiring maintainer review on all pull requests
- Added GPG commit verification section to `SECURITY.md` with key ID, fingerprint, algorithm, and keyserver fetch instructions
- Added vulnerability response SLA to `SECURITY.md`: critical issues fixed within 90 days, all others within 180 days

### Documentation

- Added `Image Signing and Release Metadata` section to `docs/security.rst` with `cosign verify` and `cosign verify-attestation` commands and OIDC identity constraints
- Documented TUF-on-CI metadata repository and its role in isolating TUF lifecycle from application source
- Added `docs/releasing.rst` with end-to-end release process: changelog preparation, `gh release create` with release notes, `tuf-on-ci-sign` workflow, TUF signing keys table, and instructions for delegating signing to a new maintainer
- Added `releasing` to `docs/index.rst` Development toctree
- Added `Development` section to `README.md` with `make test`, `make lint`, and `make build` commands
- Added SLSA Level 3 badge to `README.md`
- Added zero-warnings policy to `CONTRIBUTING.md`: all lint warnings must be resolved before a PR is merged

### CI

- Updated release workflow permissions: added `id-token: write` for OIDC and `contents: write` for release asset uploads
- Added `TUF_TARGET_REPO` env var and `TUF_REPO_TOKEN` secret usage to release workflow
- Added `provenance` job to release workflow generating SLSA Level 3 container provenance attestations
- Added CycloneDX SBOM generation via `anchore/sbom-action` - uploaded as a release asset and attested with cosign on every release

## [1.4.0] - 2026-04-10

### Added

- **Comprehensive test suite**: Added unit tests for config, discovery, handler, middleware, and server packages
- **Testing policy docs**: Expanded testing guidance for OpenSSF-aligned practices
- **Dependabot automation**: Added weekly updates for Go modules and GitHub Actions
- **Linting policy baseline**: Added `golangci-lint` configuration with `gosec` and `staticcheck`

### Changed

- **Go toolchain**: Upgraded project Go version to `1.26.0` in module metadata
- **Docker build toolchain**: Updated builder image to `golang:1.26-alpine`
- **Dependency update**: Bumped `golang.org/x/time` from `v0.5.0` to `v0.15.0`

### CI

- **Workflow upgrade**: Updated `actions/setup-go` from `v4` to `v6`
- **CI Go version**: Updated workflow `go-version` to `1.26.x`
- **Action dependency refresh**: Upgraded Docker and security-related GitHub Actions dependencies

### Documentation

- Added private vulnerability reporting guidance in `SECURITY.md`
- Updated build prerequisites to require Go `1.26` or later in `docs/building.rst`

## [1.3.0] - 2026-04-04

### Changed

- **License**: Switched from MIT to Apache 2.0

### CI

- Skip Go test and lint workflow on changes to `docs/**`, `**.md`, `**.rst`, and `**.txt`

## [1.2.0] - 2026-03-21

### Added

- **OCI IAM policy documentation**: Added required IAM policy statements to README and `docs/installation.rst` covering all five API call families (`read instances`, `read compartments`, `read vnic-attachments`, `read vnics`, `read tag-namespaces`) with per-call mapping table and note on `NotAuthorizedOrNotFound` behavior

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

[Unreleased]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.5.1...HEAD
[1.5.1]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.4.2...v1.5.0
[1.4.2]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.4.1...v1.4.2
[1.4.1]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/amaanx86/oci-prometheus-sd-proxy/releases/tag/v1.0.0
