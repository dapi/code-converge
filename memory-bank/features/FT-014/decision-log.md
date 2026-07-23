---
title: "FT-014: Decision Log"
doc_kind: feature
doc_function: record
purpose: "Provenance журнала решений, FPF-анализа, review-improve циклов и human gates для FT-014."
derived_from:
  - brief.md
  - ../../flows/routing.md
  - ../../flows/feature.md
  - ../../engineering/validation-profiles.md
  - ../../../README.md
  - ../../../internal/config/config.go
  - ../../../internal/runner/runner.go
  - https://github.com/dapi/code-converge/issues/14
status: active
audience: humans_and_agents
must_not_define:
  - canonical_problem_space
  - selected_solution
  - implementation_sequence
---

# FT-014: Decision Log

## Routing Record — 2026-07-23

- **Selected flow:** Feature Flow.
- **Facts:** Issue #14 requests one independently verifiable workflow-diagnostics delivery unit. It changes CLI flags, configuration resolution/display, diagnostic-data persistence, retention and opt-out behavior. The root README owns the public CLI/configuration/stdout contract; `internal/config` implements the existing four-source resolution model and `internal/runner` captures process streams without forwarding them to workflow stdout.
- **Rejected routes:** Small Change is ineligible because CLI/configuration and persistent-data behavior change and the issue expressly requires design decisions. It is not a bug, incident or behavior-preserving refactoring. The issue defines one delivery unit and supplies no evidence that an epic roadmap or multiple delivery features are required.
- **Result:** Bootstrap `FT-014` with `brief.md`; `Design required: yes`; validation profile `high-risk` because durable potentially sensitive data, permissions/redaction controls and persistent-data retention are in scope.

## D-01 — Public diagnostic-storage contract

- **Status:** accepted — 2026-07-23.
- **Prompt (FPF B.5 reasoning cycle):** Issue #14 requires exact public names and validation rules for the log directory, retention duration and opt-out flag; it also requires a session layout, concurrency behavior, disabled-retention meaning, logging-write failure semantics and redaction policy. It intentionally lists these as required decisions, not selected requirements.
- **Bounded contexts:** Public CLI/configuration; private session-record filesystem; Codex process boundary; workflow stdout. FPF strict distinction keeps the private diagnostic record separate from the stable public event stream and from the process environment, which must not be logged.
- **Available facts:**
  - The existing configuration precedence is CLI > project > user > environment > built-in default, and `code-converge config` displays effective settings and sources.
  - The issue fixes only the default root `~/.code-converge/`, default retention of 24 hours, per-run opt-out intent, private raw-stream capture, no environment/token logging, owner-only permissions where supported, best-effort bounded cleanup, and unchanged workflow stdout.
  - Existing option/file/environment naming follows `kebab-case`, `CODE_CONVERGE_*` and the same basename; existing duration-like heartbeat accepts `0` or a Go duration of at least one second.
  - No current document or issue fact ranks any candidate public names, disabled-retention semantics, write-failure policy, session layout, or redaction transformation.
- **Candidates:**
  1. **Public names:** choose names patterned as `session-log-dir` / `CODE_CONVERGE_SESSION_LOG_DIR` / `--session-log-dir`, `session-log-retention` / `CODE_CONVERGE_SESSION_LOG_RETENTION` / `--session-log-retention`, and `--no-session-log`; other names remain plausible because Issue #14 says the final opt-out name may differ.
  2. **Retention disabled value:** reject `0`; treat `0` as immediate cleanup; or treat `0` as no cleanup. Each has materially different data-retention and operator expectations.
  3. **Write failure:** fail workflow; emit a warning/diagnostic and continue; or make it configurable. Each changes the reliability/privacy trade-off and public error behavior.
  4. **Record/redaction design:** a per-session directory with isolated invocation entries; one append-only session file; or another layout. Redact no command/prompt content beyond tokens; redact defined patterns; or omit selected fields. Each trades reproducibility against confidential-data exposure and affects concurrency/collision behavior.
- **FPF selection criteria:** Preserve private diagnostics versus public workflow stdout as separate bounded contexts; choose existing naming/configuration patterns where available; prefer the smallest reversible structure that prevents concurrent ownership conflicts; and choose controls that bound durable sensitive data without reducing a successful workflow to a diagnostic failure.
- **Selection (authorized FPF decision):**
  1. Public settings are `--session-log-dir`, `CODE_CONVERGE_SESSION_LOG_DIR`, `session-log-dir`; `--session-log-retention`, `CODE_CONVERGE_SESSION_LOG_RETENTION`, `session-log-retention`; and per-run `--no-session-log`. Default directory: `~/.code-converge/session-logs`; default retention: `24h`.
  2. Retention is a Go duration of at least one second; zero and negative values are invalid. `--no-session-log`, rather than a retention sentinel, is the only no-artifact control.
  3. A record create/write/permission or cleanup failure writes a prefixed stderr diagnostic and otherwise leaves workflow control flow and exit status unchanged. It is not configurable.
  4. An enabled run owns one private `0700` directory named with UTC timestamp, PID and cryptographic random suffix. It contains atomic `0600` JSON lifecycle/invocation records ordered by a per-session sequence; cleanup considers only verified direct non-symlink session children under the effective root.
  5. Environment values are never recorded. Every persisted text field is redacted for explicit secret-bearing argument/key names, bearer credentials and documented GitHub-token prefixes; the operator warning and `--no-session-log` cover residual sensitivity of repository content and arbitrary agent output.
- **Rationale:** The names reuse the project convention and meet the exact acceptance surfaces. Rejecting zero avoids an ambiguous setting that either destroys the promised diagnostic immediately or silently permits indefinite retention; explicit opt-out has one clear meaning. Warning-only failures preserve the existing workflow's observable success/failure contract while stderr makes diagnostic loss visible. Per-session directories are the minimal isolation boundary for concurrent sessions and retention cleanup. Deterministic redaction plus omitted environments directly enforces the issue's non-logging constraint without pretending arbitrary repository/agent content is non-sensitive.
- **Rejected candidates:** A new naming family lacks a project advantage; zero-as-immediate cleanup overlaps opt-out and can defeat discoverability; zero-as-no-cleanup expands sensitive-data retention without a stated need; fatal/configurable logging errors introduce unnecessary workflow behavior variation; a shared append-only file complicates concurrent ownership and cleanup; no-redaction conflicts with the token constraint.
- **Execution authorization:** The user's end-to-end implementation instruction authorizes `AG-01` for this bounded local/PR delivery. No production, live-data or external-log-upload action is authorized.

## D-02 — Human session-log handoff

- **Status:** accepted — 2026-07-22 issue-owner handoff.
- **Decision:** When logging is enabled and a record path has been created, human progress emits exactly one initial permanent local `HH:MM:SS Session log: <path>` line. `kv` and per-run opt-out do not emit it, and it never contains session content.
- **Source:** Issue #14 owner comment `issuecomment-5051013615`.
- **Rationale:** The comment is an explicit upstream contract. It adds discoverability without extending the structured event schema or exposing raw diagnostic contents.

## Review-Improve Record

### Cycle 1 — 2026-07-23

- **Review scope:** Issue #14, the new `FT-014` bootstrap package, root README configuration/stdout contract, `internal/config`, `internal/runner`, Feature Flow, validation profile and testing policy.
- **Critical findings:** none.
- **Important findings:**
  - `I-01` — no feature package existed despite the issue's explicit Feature Flow route.
  - `I-02` — the issue leaves material public and security-sensitive design choices unresolved; creating design/plan artifacts would invent a solution.
- **FPF closure:** FPF B.5 reasoning and bounded-context/assurance checks constrained alternatives but could not select them from documented facts. `D-01` is therefore a human gate, not an inferred decision.
- **Action:** Created `README.md`, canonical `brief.md` and this decision log; recorded the Feature Flow route, verification contract and gate. No downstream artifacts were created.

### Cycle 2 — 2026-07-23

- **Review scope:** accepted `D-01` against Issue #14, existing configuration naming/precedence, process-runner capture boundary, the canonical `brief.md`, selected `design.md` and derived `implementation-plan.md`.
- **Critical findings:** none.
- **Important findings:** `I-02` is closed by the user-authorized FPF decision above; no other important inconsistency remains.
- **FPF closure:** Applied bounded contexts, existing-pattern reuse, smallest-safe ownership boundary and assurance limits to select one coherent public/configuration/storage/redaction contract.
- **Action:** Promoted the accepted decision into `brief.md`; created `design.md` and `implementation-plan.md`; updated the feature index. Implementation remains unstarted pending `AG-01`.

### Cycle 3 — 2026-07-23

- **Review scope:** complete FT-014 package and its `memory-bank/features/README.md` index entry.
- **Critical findings:** none.
- **Important findings:** `I-03` — the feature index still described FT-014 as blocked even though `D-01` had been accepted and downstream artifacts existed.
- **FPF closure:** none required; this was a stale derived status, not a new decision.
- **Action:** Updated the index and package routing text to describe the accepted FPF design and ready implementation plan. No minor findings were changed.

### Cycle 4 — 2026-07-23

- **Review scope:** complete FT-014 package: `README.md`, `brief.md`, `decision-log.md`, `design.md`, `implementation-plan.md`; related feature index; Issue #14 acceptance and existing public configuration/process-boundary facts.
- **Critical findings:** none.
- **Important findings:** none.
- **Minor findings:** none recorded; no minor-only changes made.
- **FPF closure:** none required.
- **Result:** Stopped review-improve early: canonical problem, solution, execution and evidence owners are linked and agree; no critical or important document finding remains.
