# Agent Browser Benchmark

Natural-language benchmark lane for running the same fixture tasks with
[`agent-browser`](https://github.com/vercel-labs/agent-browser) instead of the
PinchTab CLI.

## Setup

1. Read `../../skills/agent-browser/SKILL.md` first. This is your guide to the
   benchmark wrapper and command workflow.

2. Start the Docker benchmark environment and initialize a report:

```bash
cd tests/benchmark
./scripts/run-agent-browser-benchmark.sh
```

3. Use the Docker-backed wrapper for every browser action:

```bash
./scripts/ab open http://fixtures/
./scripts/ab snapshot -i -c
./scripts/ab click @e2
./scripts/ab fill @e3 "agent@benchmark.test"
```

4. Record each completed step:

```bash
./scripts/record-step.sh --type agent-browser 1 1 pass --tokens 120 48 "completed"
```

`record-step.sh` will automatically calculate the number of `agent-browser`
tool calls used since the previous recorded step by reading
`results/agent_browser_commands.ndjson`.

## Environment

- Fixtures: `http://fixtures/`
- Session name: `benchmark` by default (`AGENT_BROWSER_SESSION` overrides)
- Browser driver: Docker service `agent-browser`

## Tooling Guidance

Use `../../skills/agent-browser/SKILL.md` as the primary operating guide.
Reach for `./scripts/ab --help` only when the skill does not already answer the
question.

## Task Set

Reuse the same benchmark task groups from [AGENT_TASKS.md](./AGENT_TASKS.md)
for content extraction, search, forms, SPA state, login, e-commerce, exports,
dialogs, async flows, drag/drop, keyboard, scrolling, and iframe interaction.

The only setup difference is that Group 0 should validate the `agent-browser`
lane instead of the PinchTab server:

- 0.1 `./scripts/ab open http://fixtures/` succeeds
- 0.2 `./scripts/ab snapshot -i -c` returns interactive refs
- 0.3 session state persists across multiple `./scripts/ab ...` commands

After that, continue with the same user-facing tasks from Group 1 onward.
