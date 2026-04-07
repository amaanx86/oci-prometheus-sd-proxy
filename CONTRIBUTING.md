# Contributing

## Security

Found a security vulnerability? Do not open a GitHub issue. See [SECURITY.MD](SECURITY.MD) for responsible disclosure guidelines.

## Setup

```bash
# Download and tidy dependencies
make tidy
```

## Code Style

- Run `gofmt` or `goimports` before committing
- Run `make lint` to check for issues
- Max line length: 100 characters
- Naming: camelCase for Go identifiers, snake_case for config keys
- Write tests for new functionality

## Workflow

1. Create a feature branch:
   ```bash
   git checkout -b feature/my-feature
   ```

2. Make changes and verify:
   ```bash
   make lint
   make test
   ```

3. Commit using conventional commits:
   ```bash
   git commit -S -m "feat: add new feature"
   ```

4. Push and open a pull request:
   ```bash
   git push origin feature/my-feature
   ```

## Commit Messages

GPG-sign all commits (`git commit -S`) and follow conventional commits format:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation
- `test:` - Tests
- `refactor:` - Code refactoring
- `chore:` - Build, dependencies, etc.

To always sign commits automatically:
```bash
git config commit.gpgsign true
```

## Testing

**Policy: all new functionality and bug fixes must include automated tests.**
This requirement applies to every pull request - PRs that add code without
corresponding tests will not be merged.

Run the full test suite (with race detector and coverage):

```bash
make test
# equivalent to: go test -race -cover ./...
```

Run a single package:

```bash
go test -race -run TestFunctionName ./internal/discovery/
```

### What to test

| Area | Location | Notes |
|------|----------|-------|
| Config loading and validation | `internal/config/` | Table-driven, use `t.TempDir()` for temp files |
| Middleware (auth, logging) | `internal/middleware/` | Use `httptest.NewRequest` / `httptest.NewRecorder` |
| HTTP handler | `internal/handler/` | Mock `TargetCache` interface |
| Discovery pure functions | `internal/discovery/` | `hasTag`, `isWindows`, `buildLabels`, `sanitizeLabelKey`, `extractErrorCode` |
| Cache utilities | `internal/discovery/` | `targetSet`, `computeDelta`, `Get`, `LastError` |
| Server routing | `internal/server/` | Use `httptest.NewServer` |

OCI API-calling methods (`discoverTenancy`, `discover`, `discoverCompartment`,
`listAllInstances`, `listAllCompartments`, `getPrimaryPrivateIP`) require live
OCI credentials and are excluded from unit testing. Integration tests for these
functions belong in a separate `_integration_test.go` file and are skipped by
default with a build tag.

### Coverage

We target >= 60% statement coverage overall. The hard floor per package is:

- `internal/config` - 90%+
- `internal/middleware` - 100%
- `internal/handler` - 100%
- `internal/server` - 100%
- `internal/discovery` - 40%+ (limited by OCI SDK boundary)

## Building

```bash
# Build binary (outputs to ./bin/oci-sd-proxy)
make build

# Build Docker image
make docker

# Run locally (requires SERVER_TOKEN)
SERVER_TOKEN=your-token make run
```

## Pre-commit Hooks

Hooks run automatically on `git commit`:

- Trailing whitespace trimming
- YAML validation
- Merge conflict detection
- Go formatting and linting

If a hook fails, fix the reported issues and retry the commit.

## Questions

Open an issue or discussion in the repository.
