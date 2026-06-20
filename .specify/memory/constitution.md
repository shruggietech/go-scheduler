<!--
SYNC IMPACT REPORT
==================
Version change: (template) → 1.0.0
Bump rationale: Initial ratification of the project constitution (MAJOR baseline).

Modified principles: N/A (initial adoption)
Added principles:
  - I. Code Quality
  - II. Testing Standards (NON-NEGOTIABLE)
  - III. User Experience Consistency
  - IV. Performance Requirements
Added sections:
  - Engineering Constraints
  - Development Workflow & Quality Gates
  - Governance

Templates requiring updates:
  ✅ .specify/templates/plan-template.md (Constitution Check gates align with v1.0.0)
  ✅ .specify/templates/spec-template.md (no mandatory-section changes required)
  ✅ .specify/templates/tasks-template.md (principle-driven task categories covered)
  ✅ CLAUDE.md (references constitution as governing document)

Deferred TODOs: None
-->

# go-scheduler Constitution

## Core Principles

### I. Code Quality

Code MUST be correct, readable, and idiomatic before it is considered complete.

- All Go code MUST pass `gofmt`, `go vet`, and the project linter (`golangci-lint`) with
  zero warnings; CI MUST reject any change that does not.
- Public packages, exported types, and exported functions MUST carry doc comments that
  explain intent and contract, not restate the signature.
- Errors MUST be handled explicitly: wrap with context using `fmt.Errorf("...: %w", err)`,
  never silently discard with `_`, and never `panic` in library code for recoverable
  conditions.
- Functions MUST do one thing; cyclomatic complexity SHOULD stay low and any function the
  linter flags MUST be refactored or justified in review.
- Concurrency primitives (goroutines, channels, locks) MUST have a documented ownership and
  lifecycle; every goroutine MUST have a defined termination path and the race detector
  (`go test -race`) MUST pass.

**Rationale**: A scheduler is long-running, concurrent infrastructure. Defects in error
handling or goroutine lifecycle leak resources and corrupt timing guarantees, so quality is
enforced mechanically rather than left to discretion.

### II. Testing Standards (NON-NEGOTIABLE)

Tests are written alongside or before the code they verify, and the suite is the source of
truth for correctness.

- Every behavioral change MUST ship with tests; bug fixes MUST include a regression test that
  fails before the fix and passes after.
- Unit tests MUST cover scheduling logic, time/clock handling, and error paths. Time MUST be
  injected through an abstraction (e.g., a `Clock` interface) — tests MUST NOT depend on real
  wall-clock `time.Sleep` for deterministic assertions.
- Integration tests MUST cover job persistence, recovery after restart, and concurrent
  job execution.
- All tests MUST run under `go test -race`; flaky tests MUST be fixed or quarantined with a
  tracking issue, never ignored.
- Coverage on core scheduling packages MUST be ≥ 80%; new code MUST NOT lower package
  coverage. CI MUST enforce these gates.

**Rationale**: Correct timing and reliable recovery are the product's core promise. Without
deterministic, race-checked tests these guarantees cannot be verified, so testing discipline
is non-negotiable.

### III. User Experience Consistency

Every interface the project exposes — CLI, configuration, logs, and API — MUST behave
predictably and uniformly.

- The CLI MUST follow a consistent verb-noun command structure, support both
  human-readable and `--json` output, write results to stdout and errors/diagnostics to
  stderr, and return conventional exit codes (0 success, non-zero failure).
- Configuration MUST have a single documented schema with sensible defaults; invalid
  configuration MUST fail fast at startup with a clear, actionable message naming the field.
- Time inputs and outputs MUST use a consistent format (RFC 3339) and timezone handling MUST
  be explicit; durations MUST use Go duration syntax consistently.
- Error messages MUST be actionable: state what failed, why, and what the user can do.
- Logging MUST be structured and consistent across components (consistent field names,
  levels, and correlation/job identifiers).

**Rationale**: Operators interact with a scheduler under time pressure during incidents.
Consistent, self-explanatory interfaces reduce operator error and the cost of recovery.

### IV. Performance Requirements

The scheduler MUST be efficient and meet stated timing and resource budgets.

- Scheduling decisions MUST be measured: job dispatch latency (scheduled time → execution
  start) MUST stay within a documented budget (default target: p99 < 100ms under nominal
  load) and the budget MUST live next to the code it governs.
- Performance-sensitive changes MUST include benchmarks (`go test -bench`); changes MUST NOT
  regress an existing benchmark by more than 10% without explicit, recorded justification.
- The system MUST NOT leak goroutines or memory under sustained load; resource usage MUST be
  bounded and verified for the supported job-count target.
- Hot paths MUST avoid unnecessary allocations and unbounded data structures; algorithmic
  complexity of scheduling operations MUST be documented.
- Premature optimization is rejected: optimizations MUST be justified by a benchmark or
  profile, not by intuition.

**Rationale**: A scheduler that drifts, stalls, or leaks under load fails silently and
erodes trust. Performance is therefore a measured, budgeted, and continuously verified
property rather than an afterthought.

## Engineering Constraints

- Language and tooling: Go (latest stable minor release), managed with Go modules. Avoid
  third-party dependencies where the standard library suffices; every new dependency MUST be
  justified in review and pass a license check.
- Supported platforms: the project MUST build and pass tests on Linux and Windows.
- Backward compatibility: persisted job state and configuration schemas MUST migrate
  forward; breaking changes to either require a MAJOR version bump and a documented migration
  path.
- Security: secrets MUST NOT be logged; inputs from configuration and APIs MUST be validated
  at the boundary.

## Development Workflow & Quality Gates

- Every change lands via pull request; no direct pushes to the default branch.
- CI MUST pass before merge and MUST enforce: `gofmt`/`go vet`, linter, `go test -race`,
  coverage thresholds, and benchmark regression checks for performance-sensitive packages.
- Code review MUST verify compliance with all four core principles. A reviewer MUST block any
  change that weakens a principle without recorded justification.
- Any deviation from a principle MUST be documented in the PR description under a
  "Complexity / Deviation" note explaining why a simpler compliant approach was rejected.

## Governance

This constitution supersedes ad-hoc practices and conventions. When a technical decision
conflicts with these principles, the principles win unless an explicit, recorded amendment
changes them.

- **Authority**: All PRs, reviews, and design documents MUST verify compliance with the four
  core principles and the constraints above. Reviewers act as the enforcement mechanism;
  CI gates act as the automated backstop.
- **Guiding decisions**: Technical and implementation choices (architecture, dependencies,
  data structures, interface design) MUST be evaluated against the principles. The default
  bias is the simplest design that satisfies all four; added complexity MUST be justified in
  writing against a named principle (typically Performance or Testing) and recorded in the
  PR.
- **Amendment procedure**: Amendments require (1) a written proposal describing the change and
  rationale, (2) review approval, and (3) a synchronized update of dependent templates and
  guidance docs. The Sync Impact Report at the top of this file MUST be updated on every
  amendment.
- **Versioning policy**: This constitution is versioned with semantic versioning.
  MAJOR = backward-incompatible governance/principle removals or redefinitions;
  MINOR = a new principle/section or materially expanded guidance;
  PATCH = clarifications and non-semantic refinements.
- **Compliance review**: Compliance is checked at every PR. Periodically (at minimum each
  release), maintainers MUST review whether the principles still reflect reality and amend
  rather than let practice silently drift.
- **Runtime guidance**: Use `CLAUDE.md` and `.specify/` templates for day-to-day development
  guidance; those documents MUST stay consistent with this constitution.

**Version**: 1.0.0 | **Ratified**: 2026-06-19 | **Last Amended**: 2026-06-19
