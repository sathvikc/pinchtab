---
name: agent-browser
description: "Use this skill when a benchmark or local automation task should drive the browser through the agent-browser CLI instead of PinchTab. In this repo, prefer the benchmark wrapper `./tests/benchmark/scripts/ab` so commands execute inside the shared Docker benchmark environment and tool calls are logged."
---

# Browser Automation with Agent Browser

For benchmark runs in this repo, do not call `agent-browser` directly. Use the
wrapper:

```bash
./tests/benchmark/scripts/ab ...
```

That wrapper:

- executes `agent-browser` inside the benchmark Docker service
- preserves the shared browser session across commands
- logs every tool call into `tests/benchmark/results/agent_browser_commands.ndjson`

Use this skill when the benchmark should run the same fixture tasks as the
PinchTab lane, but with `agent-browser` as the browser-control surface.

## Benchmark Environment

- Shared fixture site: `http://fixtures/`
- Wrapper command: `./tests/benchmark/scripts/ab`
- Default session name: `benchmark`
- Report recorder: `./tests/benchmark/scripts/record-step.sh --type agent-browser ...`

Before running tasks, initialize the benchmark lane:

```bash
cd tests/benchmark
./scripts/run-agent-browser-benchmark.sh
```

## Core Workflow

Follow this pattern:

1. Open or navigate the target page with `./tests/benchmark/scripts/ab open <url>`
2. Inspect the page with `./tests/benchmark/scripts/ab snapshot -i -c`
3. Use fresh refs such as `@e2`, `@e7`, `@e11` for actions
4. Re-snapshot after any action that changes the DOM
5. Record the finished benchmark step with `record-step.sh`

Rules:

- Prefer `snapshot -i -c` to find actionable elements
- Prefer `@eN` refs from the latest snapshot over brittle selectors
- Re-snapshot after navigation, form submit, modal open, tab change, accordion expand, or SPA update
- Use the same session for the whole benchmark run unless a task explicitly needs a reset

## Essential Commands

### Navigation and observation

```bash
./tests/benchmark/scripts/ab open http://fixtures/
./tests/benchmark/scripts/ab back
./tests/benchmark/scripts/ab forward
./tests/benchmark/scripts/ab reload
./tests/benchmark/scripts/ab snapshot
./tests/benchmark/scripts/ab snapshot -i -c
./tests/benchmark/scripts/ab snapshot -i --urls
./tests/benchmark/scripts/ab get text
./tests/benchmark/scripts/ab get title
./tests/benchmark/scripts/ab get url
```

### Interaction

```bash
./tests/benchmark/scripts/ab click @e2
./tests/benchmark/scripts/ab fill @e4 "agent@benchmark.test"
./tests/benchmark/scripts/ab type @e5 "hello"
./tests/benchmark/scripts/ab press Enter
./tests/benchmark/scripts/ab hover @e7
./tests/benchmark/scripts/ab check @e8
./tests/benchmark/scripts/ab uncheck @e8
./tests/benchmark/scripts/ab select @e9 "uk"
./tests/benchmark/scripts/ab drag @e10 @e11
./tests/benchmark/scripts/ab scroll down 800
./tests/benchmark/scripts/ab scrollintoview @e12
./tests/benchmark/scripts/ab wait @e13
./tests/benchmark/scripts/ab wait 1000
```

### Exports and debugging

```bash
./tests/benchmark/scripts/ab screenshot /tmp/page.png
./tests/benchmark/scripts/ab pdf /tmp/page.pdf
./tests/benchmark/scripts/ab console
./tests/benchmark/scripts/ab errors
./tests/benchmark/scripts/ab eval "document.title"
```

## Selector Guidance

`agent-browser` accepts:

- refs from snapshot output, like `@e2`
- CSS selectors, like `#submit` or `.btn-primary`
- XPath selectors, like `//button[@type="submit"]`

Prefer refs first. They are the most stable option inside the benchmark.

## Benchmark Reporting

After completing a benchmark step, record it:

```bash
./tests/benchmark/scripts/record-step.sh \
  --type agent-browser \
  --tokens 120 48 \
  1 1 pass \
  "opened page and extracted content"
```

`record-step.sh` automatically derives `tool_calls` for the step from the
wrapper log unless you override it with `--tool-calls <n>`.

## Task Entry Point

Use this skill together with:

- `tests/benchmark/AGENT_BROWSER_TASKS.md`

That file contains the benchmark tasks; this skill tells you how to operate the
browser tool surface correctly.
