# Production hardening TODO (prioritized)

This file lists prioritized, minimal-impact engineering improvements to increase reliability, maintainability, and developer experience for the codebase.

- [ ] Add unit tests for core packages
  - Target: internal/brain, internal/reactor, internal/prompt, internal/model and any public pkg APIs.
  - First step: add a small table-driven test per package with mocks/fakes where appropriate and run with `go test ./...`.

- [ ] Add integration/E2E tests
  - Target: a lightweight end-to-end test that runs the CLI binary (cmd/vibeaura) in a sandboxed environment and exercises a simple brain->reactor interaction.
  - First step: add an integration-tests/ folder with a single reproducible scenario and a script to run it.

- [ ] Add CI workflow (GitHub Actions)
  - Tasks: run `go test ./...`, `go vet`, `golangci-lint`, `go mod tidy` and build the binary on push/PR.
  - First step: add `.github/workflows/ci.yml` with minimal steps and cache modules.

- [ ] Enable static analysis and linters
  - Tools: golangci-lint (staticcheck, gofmt enforcement), go vet.
  - First step: add `.golangci.yml` with default rule set and a CI step that fails on issues.

- [ ] Add a Makefile with common targets
  - Targets: `build`, `test`, `lint`, `fmt`, `vet`, `ci`, `integration`.
  - First step: create a simple Makefile that wraps the above commands for consistent developer UX.

- [ ] Add ARCHITECTURE.md
  - Provide a short system overview: modules, responsibility boundaries, typical interaction flow (CLI -> brain -> reactor -> tooling), and key extension points.
  - First step: create a one-page document in repo root summarizing packages in `go.work`.

- [ ] Add developer documentation and examples
  - Files: `docs/local-development.md`, `docs/examples.md` and a CONTRIBUTING snippet with setup steps.
  - First step: document how to run the app, run tests, and add a new vibe/plugin.

- [ ] Pre-commit hooks and repository hygiene
  - Tools: husky/lefthook or simple git hooks to run `gofmt`, `golangci-lint run`, and quick tests on staged files.
  - First step: add a `.githooks/pre-commit` that runs `gofmt -w` and `go vet`.

- [ ] Add lightweight telemetry/logging patterns
  - Use structured logs with levels and a small, opt-in telemetry stub (no secrets) to observe core loop behavior for debugging.
  - First step: standardize on a logger interface and add a small example in `cmd/vibeaura`.

- [ ] Release/versioning guidance
  - Document tagging policy, changelog format, and a minimal `release` Makefile target.
  - First step: add `RELEASE.md` with recommended steps for version bump and tagging.

- [ ] Add a sandbox or local integration helper
  - Provide a Dockerfile or script that runs the app in isolation for integration tests and developer experimentation.
  - First step: add `dev/docker/Dockerfile` (minimal) and `dev/run-local.sh`.

- [ ] Secrets & config hygiene
  - Add `.env.example`, document secret handling, and recommend using OS keyring or CI secrets for tokens.
  - First step: add `.env.example` and `docs/secrets.md` with best practices.

- [ ] Templates and tests for new "vibes" (plugins)
  - Provide a template directory with a sample vibe, test harness, and a checklist for submissions.
  - First step: add `vibes/template/` with README and test.

Notes and priorities
- Quick wins (days): Makefile, one unit test per core package, ARCHITECTURE.md, Makefile `test` target.
- Medium effort (1-2 weeks): CI workflow, golangci-lint configuration, integration test scaffold, pre-commit hooks.
- High value but later: sandbox Dockerfile, telemetry, release automation and comprehensive test coverage.

Keep changes minimal and incremental: prefer adding small, well-scoped tests and automation first, then iterate on linting and CI enforcement.
