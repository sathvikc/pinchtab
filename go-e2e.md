# Go E2E Runner Plan

> Historical implementation plan. The canonical operating contract now lives in
> `TESTING.md` and `tests/e2e/README.md`.

## Goal

Extend `tests/tools/runner` so one Go command can support both the existing benchmark/optimization runner and a deterministic e2e runner.

The e2e runner replaces the old shell e2e orchestration for host-side decisions. Shell remains only as the deterministic in-container scenario executor.

## Current Status

Implemented through host orchestration and explicit scenario selection:

- `tests/tools/runner/main.go` dispatches benchmark commands and `runner e2e`.
- `internal/bench` owns the existing benchmark/optimization runner.
- `internal/e2e` owns suite parsing, Docker Compose lifecycle, suite matrix, log mode, captured output logs, summary/report generation, failure artifacts, and exact scenario file selection.
- `tests/e2e/run.sh` accepts explicit `scenario=<file>` arguments and acts as the in-container executor. It has no scenario discovery, filter fallback, or durable report writing.
- The old `scripts/e2e.sh` host dispatcher and e2e aliases are removed.
- Smoke suites are wired through the Go runner. Infra smoke coverage includes orchestrator, security, autosolver, dashboard, lifecycle, and host Docker smoke checks.

## Target Commands

Preferred command surface:

```bash
go run ./tests/tools/runner e2e --suite basic
go run ./tests/tools/runner e2e --suite extended
go run ./tests/tools/runner e2e --suite smoke
go run ./tests/tools/runner e2e --suite infra-extended --filter orchestrator
go run ./tests/tools/runner e2e --suite api --filter tabs
go run ./tests/tools/runner e2e --suite infra-extended --filter security
```

Backward-compatible benchmark commands must keep working:

```bash
go run ./tests/tools/runner --lane pinchtab
go run ./tests/tools/runner --lane agent-browser
go run ./tests/tools/runner step-end ...
go run ./tests/tools/runner record-step ...
go run ./tests/tools/runner verify-step ...
```

## Suite Naming

Use coverage-level names for the new e2e runner:

| Suite | Meaning |
|---|---|
| `basic` | API basic + CLI basic + infra basic |
| `extended` | API extended + CLI extended + infra extended + plugin, excluding smoke-only scenarios |
| `smoke` | expensive topology, lifecycle, and manually targeted smoke coverage |
| `api`, `cli`, `infra`, `plugin` | existing single basic suite |
| `api-extended`, `cli-extended`, `infra-extended` | existing single extended suite |

## Package Layout

Refactor `tests/tools/runner` into a thin dispatcher plus internal packages:

```text
tests/tools/runner/
  main.go
  internal/
    bench/
      args.go
      loop.go
      prompt.go
      setup.go
      report.go
      stepend.go
      recordstep.go
      verifystep.go
      ...
    e2e/
      args.go
      runner.go
      suite.go
      shell_delegate.go
      compose.go
      report.go
      timing.go
      ...
    shared/
      output.go
      retry.go
      shell.go
      ...
```

The first refactor should move code mechanically. Avoid changing benchmark behavior while moving files.

## Benchmark And Baseline Pattern

Use the benchmark/optimization flow as the model:

- `./dev bench ...` is a thin entrypoint into `go run ./tests/tools/runner`.
- The Go runner owns argument parsing, setup, active report pointers, progress summaries, timing, and finalization.
- Shell remains where it is the deterministic executor or domain wrapper:
  - benchmark wrappers: `tests/tools/scripts/pt`, `tests/tools/scripts/ab`
  - deterministic baseline: `tests/tools/scripts/baseline.sh`
- The deterministic baseline records into the same Go-managed report contract via `./scripts/runner record-step` and `verify-step`.

For e2e, aim for the same split:

- Go owns the host-side suite runner, Docker Compose orchestration, suite selection, captured output logs, summaries, markdown reports, failure artifacts, and dev/CI command surface.
- Bash scenarios remain deterministic executable assets for now, like `baseline.sh`.
- Container-side shell should become a small scenario executor, not the main orchestrator.
- Do not require porting every scenario to Go before gaining structure and speed wins.

## Main Dispatcher

`main.go` should only route commands:

```text
runner e2e ...           -> internal/e2e.Run
runner bench ...         -> internal/bench.Run
runner step-end ...      -> internal/bench.RunStepEnd
runner record-step ...   -> internal/bench.RunRecordStep
runner verify-step ...   -> internal/bench.RunVerifyStep
runner --lane ...        -> internal/bench.Run, for compatibility
```

Rules:

- If the first arg is `e2e`, use the e2e runner.
- If the first arg is `bench`, strip it and use the benchmark runner.
- If the first arg is `step-end`, `record-step`, or `verify-step`, preserve existing behavior.
- If args contain `--lane` without a subcommand, preserve existing benchmark behavior.

## Phase 1: Mechanical Package Split

Move current benchmark runner code from root `package main` into `internal/bench`.

Keep tests passing and command behavior unchanged.

Likely moved files:

- `args.go`
- `loop.go`
- `prompt.go`
- `setup.go`
- `report.go`
- `summary.go`
- `runner.go`
- `anthropic.go`
- `openai.go`
- `fake.go`
- `recordstep.go`
- `stepend.go`
- `verifystep.go`
- related tests

Likely shared files:

- `output.go`
- `retry.go`
- `shell.go`

Acceptance:

```bash
go test ./tests/tools/runner/...
go run ./tests/tools/runner --provider fake --lane pinchtab --dry-run
go run ./tests/tools/runner bench --provider fake --lane pinchtab --dry-run
```

## Phase 2: E2E Runner Entry Point

Add `internal/e2e` and resolve filters in Go before the container command is built:

```bash
go run ./tests/tools/runner e2e --suite infra-extended --filter orchestrator
```

becomes:

```bash
docker compose ... run --rm -e E2E_HELPER=api -e E2E_SCENARIO_DIR=scenarios/infra ... runner-api /bin/bash /e2e/run.sh scenario=orchestrator-extended.sh
```

Acceptance:

```bash
go run ./tests/tools/runner e2e --suite api --filter browser
go run ./tests/tools/runner e2e --suite basic
```

## Phase 3: Smoke Suite Support

Add smoke scenario support to the Go runner and the thin container executor.

Update `tests/e2e/run.sh` to understand:

```text
*-basic.sh
*-extended.sh
*-smoke.sh
```

Expected behavior:

- basic suites include only `*-basic.sh`
- extended suites include `*-basic.sh` + `*-extended.sh`
- smoke suites include `*-smoke.sh`
- extended excludes smoke

Acceptance:

```bash
go run ./tests/tools/runner e2e --suite smoke
go run ./tests/tools/runner e2e --suite smoke --filter orchestrator
```

## Phase 4: Move First Smoke Candidates

Move the highest-cost e2e checks into smoke files.

Start with:

- `tests/e2e/scenarios/infra/orchestrator-smoke.sh`
- `tests/e2e/scenarios/infra/security-smoke.sh`

Candidates:

- orchestrator ports, isolation, and cleanup
- real child instance launch/stop/relaunch flow
- instance-scoped security policy allowed-domain widening
- wildcard instance-scoped policy widening

Keep or backfill lower-level coverage before removing checks from extended.

Acceptance:

```bash
go run ./tests/tools/runner e2e --suite extended
go run ./tests/tools/runner e2e --suite smoke --filter orchestrator
go run ./tests/tools/runner e2e --suite smoke --filter security
```

## Phase 5: Go Host Orchestration

Move host orchestration into `internal/e2e`.

Responsibilities to move into Go first:

- Docker Compose command resolution and stack lifecycle
- shared-stack setup and teardown
- suite matrix for `basic`, `extended`, `smoke`, and single-suite runs
- filter/extra/test propagation
- logs mode and failure artifact surfacing
- suite output capture and report file naming
- timing and final suite summary/report generation

This should still call the existing container scenario executor initially:

```bash
docker compose ... run --rm -e E2E_HELPER=api -e E2E_SCENARIO_DIR=scenarios/api runner-api /bin/bash /e2e/run.sh scenario=...
docker compose ... run --rm -e E2E_HELPER=cli -e E2E_SCENARIO_DIR=scenarios/cli runner-cli /bin/bash /e2e/run.sh scenario=...
```

Acceptance:

```bash
go run ./tests/tools/runner e2e --suite basic --dry-run
go run ./tests/tools/runner e2e --suite extended --dry-run
go run ./tests/tools/runner e2e --suite infra-extended --filter orchestrator --dry-run
```

`--dry-run` should print the compose operations and container commands that would run, similar to the benchmark runner dry-run plan.

## Phase 6: Container Scenario Executor

Refactor `tests/e2e/run.sh` after host orchestration is in Go.

Target shape:

- Go selects scenario files and passes an explicit ordered list to the container.
- Bash still sources helpers and scenario scripts so existing `start_test`, `end_test`, and assertion counters continue to work.
- `end_test` emits a structured result event immediately; final totals and summaries are printed by Go.
- `run.sh` accepts only the executor contract: `scenario=<file>` plus execution metadata from the Go runner.
- Add smoke suffix support here: `*-basic.sh`, `*-extended.sh`, `*-smoke.sh`.

This mirrors `baseline.sh`: deterministic shell execution, Go-managed run/report orchestration.

## Phase 7: Native Go E2E Scenarios

Only after the shell executor is thin and stable, begin porting selected scenarios into native Go.

Start with low-risk deterministic API tests:

- health
- fixtures server check
- simple navigate
- tabs list
- simple text/snapshot
- simple security gates

Avoid porting slow topology tests first. Those should stay smoke and validate the smoke lane.

Native Go e2e scenario support should include:

- one reusable HTTP client with keepalive
- typed request helpers
- typed JSON assertions
- per-test timing
- structured reports
- cleanup hooks
- bounded parallelism for independent tests
- scenario tags: `basic`, `extended`, `smoke`, `slow`, `topology`, `lifecycle`

## Phase 8: Reporting

The Go e2e runner now owns durable summary and markdown report generation from shell-emitted structured result lines:

- `end_test` prints one `E2E_RESULT` line when each test finishes.
- The Go runner tees or captures container output into `output-*.log`.
- The Go runner parses `E2E_RESULT` lines, prints the final human summary, and writes the suite `summary-*.txt` and `report-*.md`.
- If a suite fails before a failed structured result line is emitted, the Go runner writes a failed fallback row so CI and humans still get a report artifact.
- Compose service logs are captured by the Go runner on failure.

Future report formats can be added without touching shell scenarios:

- machine JSON report
- optional JUnit XML for CI

Suggested aggregate files:

```text
tests/e2e/results/
```

Suggested files:

```text
report-e2e-basic.md
report-e2e-basic.json
report-e2e-extended.md
report-e2e-extended.json
report-e2e-smoke.md
report-e2e-smoke.json
```

## Phase 9: Dev And CI Integration

Update `./dev` to route e2e through the Go runner:

```bash
./dev e2e basic
./dev e2e extended
./dev e2e smoke
./dev e2e infra-extended --filter orchestrator
```

Update CI only after local behavior is stable.

Suggested final CI mapping:

- PR: `go run ./tests/tools/runner e2e --suite basic`
- Manual extended: `go run ./tests/tools/runner e2e --suite extended`
- Manual or scheduled smoke: `go run ./tests/tools/runner e2e --suite smoke`
- Release: `extended`, with smoke visible or blocking depending on release policy

## Risk Controls

- Preserve existing benchmark runner behavior before adding e2e behavior.
- Keep shell delegation as the first e2e milestone.
- Do not port all shell scenarios at once.
- Do not remove slow tests from extended until they exist in smoke or have lower-level coverage.
- Treat existing staged and unstaged e2e changes as user changes during implementation.

## Recommended First Milestone

1. Split current `tests/tools/runner` into `internal/bench` behind a thin `main.go`.
2. Add `internal/e2e` with host orchestration and explicit scenario selection.
3. Support:

```bash
go run ./tests/tools/runner e2e --suite basic
go run ./tests/tools/runner e2e --suite extended
go run ./tests/tools/runner e2e --suite infra-extended --filter orchestrator
```

4. Keep `go run ./tests/tools/runner --lane pinchtab --dry-run` working unchanged.
