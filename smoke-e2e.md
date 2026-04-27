# Smoke E2E Plan

> Historical implementation plan. The canonical operating contract now lives in
> `TESTING.md` and `tests/e2e/README.md`.

## Goal

Create a proper slow smoke lane so default e2e stays fast, while expensive topology and lifecycle tests remain easy to run deliberately.

## Target Shape

Keep three layers:

| Layer | Purpose | Command |
|---|---|---|
| Fast gate | PR-safe, common regressions | `go run ./tests/tools/runner e2e --suite basic` |
| Core extended | Broad coverage without costly topology tests | `go run ./tests/tools/runner e2e --suite extended` |
| Smoke | Slow real-world and topology checks | `go run ./tests/tools/runner e2e --suite smoke` |

Expose narrower smoke commands as needed:

```bash
go run ./tests/tools/runner e2e --suite smoke-orchestrator
go run ./tests/tools/runner e2e --suite smoke-security
go run ./tests/tools/runner e2e --suite smoke-lifecycle
go run ./tests/tools/runner e2e --suite smoke-docker
```

## Phase 1: Add First-Class Smoke Support

Update the Go e2e runner so it selects a third scenario suffix:

```text
*-basic.sh
*-extended.sh
*-smoke.sh
```

Expected behavior:

| Suite | Scenario Files |
|---|---|
| `api`, `cli`, `infra` | `*-basic.sh` |
| `api-extended`, `cli-extended`, `infra-extended` | `*-basic.sh` + `*-extended.sh`, excluding smoke |
| `*-smoke` | smoke scenarios only, plus required setup |
| `basic` | API basic + CLI basic + infra basic |
| `extended` | API extended + CLI extended + infra extended + plugin, excluding smoke |

Expose these directly in `internal/e2e`:

```bash
smoke
smoke-orchestrator
smoke-security
smoke-lifecycle
```

## Phase 2: Move Obvious Slow Candidates

### Orchestrator Topology Smoke

Move from `tests/e2e/scenarios/infra/orchestrator-extended.sh`:

- ports, isolation, and cleanup
- real launch/stop/relaunch multi-instance flow
- bridge attach proxy, if we want it in the same topology smoke

New file:

```text
tests/e2e/scenarios/infra/orchestrator-smoke.sh
```

### Security Child-Instance Smoke

Move or reduce from `tests/e2e/scenarios/infra/security-extended.sh`:

- instance-scoped `allowedDomains`
- wildcard scoped policy

New file:

```text
tests/e2e/scenarios/infra/security-smoke.sh
```

### Lifecycle And Timing Smoke

Move from `tests/e2e/scenarios/api/tabs-autoclose-extended.sh` and `tests/e2e/scenarios/api/tabs-extended.sh`:

- auto-close full matrix
- LRU eviction matrix
- handoff timeout auto-expiry

New file:

```text
tests/e2e/scenarios/api/tabs-smoke.sh
```

## Phase 3: Backfill Unit And Integration Coverage

Before removing slow checks from default extended, make sure lower-level tests cover the logic.

Already covered reasonably well:

- Security policy merge: `internal/orchestrator/handlers_instances_test.go`
- Orchestrator launch/stop/ports: `internal/orchestrator/orchestrator_test.go`
- Bridge proxy auth: `internal/orchestrator/proxy_test.go`
- Auto-close timer mechanics: `internal/bridge/tab_autoclose_test.go`

Likely backfill needed:

- LRU eviction ordering without sleeps
- child instance security policy enforcement boundary, mocked at orchestrator or handler level
- multi-instance tab locator behavior without launching real Chrome

## Phase 4: Add A Real All-Smoke Command

Implemented as:

```bash
./dev e2e smoke
```

This routes through the Go e2e runner and runs the smoke tier plus host Docker smoke checks:

```bash
go run ./tests/tools/runner e2e --suite smoke
```

Docker-only smoke remains available as:

```bash
./dev e2e smoke-docker
```

which maps to the runner's host Docker smoke bundle.

## Phase 5: CI Layout

Add or modify a workflow:

```text
.github/workflows/ci-smoke.yml
```

Suggested triggers:

- `workflow_dispatch`
- optional nightly or weekly `schedule`
- optional release preflight

Make it blocking only for release, or at least visible in release approval. For PRs, keep it manual unless touched files are specifically in orchestrator, security, or lifecycle areas.

## Phase 6: Metrics And Acceptance

Before and after, compare:

```bash
go run ./tests/tools/runner e2e --suite extended
go run ./tests/tools/runner e2e --suite smoke
go run ./tests/tools/runner e2e --suite basic
```

Acceptance criteria:

- `go run ./tests/tools/runner e2e --suite basic` is unchanged or faster.
- `go run ./tests/tools/runner e2e --suite extended` no longer runs real multi-instance topology or lifecycle smoke.
- `go run ./tests/tools/runner e2e --suite smoke` runs all moved expensive checks.
- reports clearly distinguish `core` from `smoke`.
- no slow test disappears without either unit coverage or smoke coverage.

## Recommended Starting Point

Start with orchestrator and security smoke first. They are the cleanest split and should give the most obvious speed win without changing normal product coverage semantics.
