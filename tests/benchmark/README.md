# PinchTab Benchmark

Structured benchmarks to measure AI agent performance with PinchTab and other
browser-control surfaces against the same fixture suite.

## Quick Start

```bash
cd tests/benchmark

# PinchTab lane
./scripts/run-optimization.sh

# agent-browser lane
./scripts/run-agent-browser-benchmark.sh
# Then run tasks from AGENT_BROWSER_TASKS.md with ./scripts/ab
./scripts/finalize-report.sh
```

## MANDATORY: Docker Environment

**The benchmark MUST run against Docker.** Do not use a local pinchtab server.

Reasons:
- Reproducible: Same environment every run
- Clean state: No leftover profiles, instances, or sessions
- Latest build: Builds from current source
- Isolated: No interference from local config

If Docker build fails or is skipped, the benchmark is **INVALID**.

## Files

| File | Purpose |
|------|---------|
| `../../skills/pinchtab/SKILL.md` | PinchTab skill (same as shipped product) |
| `../../skills/agent-browser/SKILL.md` | `agent-browser` skill for the benchmark wrapper |
| `BASELINE_TASKS.md` | Standalone task list (same as skill) |
| `AGENT_BROWSER_TASKS.md` | Equivalent task lane for `agent-browser` |
| `scripts/run-optimization.sh` | Initialize PinchTab benchmark reports |
| `scripts/run-agent-browser-benchmark.sh` | Start fixtures + `agent-browser` and initialize a fresh report |
| `scripts/ab` | Docker-backed `agent-browser` wrapper with tool-call logging |
| `scripts/record-step.sh` | Record step results with token counts and tool-call counts |
| `scripts/finalize-report.sh` | Generate final summary report |
| `config/pinchtab.json` | PinchTab configuration |
| `agent-browser/Dockerfile` | `agent-browser` benchmark image |
| `docker-compose.yml` | Docker environment definition |
| `results/` | Output directory for reports |

## Execution

The benchmark is designed to run in a fresh agent context:

1. Initialize the relevant benchmark lane
2. Execute the natural-language tasks with the target browser surface
3. Record each step's input/output tokens
4. Let the harness count browser/tool calls where possible

This measures the **real cost** of using a browser tool with an agent, including:
- Context loading overhead
- Per-task token usage
- Browser/tool-call count
- Total benchmark cost

## Environment

The benchmark runs PinchTab in Docker with:

- **Port**: 9867
- **Token**: `benchmark-token`
- **Stealth**: Full (for protected sites)
- **Headless**: Yes
- **Multi-instance**: Enabled (2 instances)

## Token Tracking

Every step must track token usage:

```bash
./scripts/record-step.sh <group> <step> <pass|fail|skip> <input_tokens> <output_tokens> "notes"
```

Example:
```bash
./scripts/record-step.sh 1 1 pass 150 45 "Navigation completed in 1.2s"
./scripts/record-step.sh 2 3 fail 200 80 "Element not found"
```

Token counts should come from your model's API response:
- **Anthropic**: `usage.input_tokens`, `usage.output_tokens`
- **OpenAI**: `usage.prompt_tokens`, `usage.completion_tokens`
- **Gemini**: `usageMetadata.promptTokenCount`, `usageMetadata.candidatesTokenCount`

## Reports

Reports are generated in `results/`:

- `benchmark_YYYYMMDD_HHMMSS.json` - Raw JSON data
- `benchmark_YYYYMMDD_HHMMSS_summary.md` - Human-readable summary
- `agent_browser_commands.ndjson` - `agent-browser` command log for tool-call attribution

### Example Summary

```
# PinchTab Benchmark Results

## Results
| Metric | Value |
|--------|-------|
| Steps Passed | 30 |
| Steps Failed | 2 |
| Pass Rate | 93.7% |

## Token Usage
| Metric | Value |
|--------|-------|
| Total Tokens | 4,523 |
| Estimated Cost | $0.0158 |
```

## Running Programmatically

For automated benchmarks, you can:

1. Parse `BASELINE_TASKS.md` for curl commands
2. Execute each command
3. Parse responses for pass/fail
4. Call `scripts/record-step.sh` with results
5. Run `scripts/finalize-report.sh`

## Reproducibility

For consistent results:

1. Always start with a fresh Docker-backed benchmark lane
2. Use the same model/temperature for comparisons
3. Run benchmarks at similar times (site load varies)
4. Record exact PinchTab version from `/version` endpoint
5. Clear browser state between full benchmark runs
