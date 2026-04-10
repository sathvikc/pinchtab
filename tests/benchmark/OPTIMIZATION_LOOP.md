# PinchTab Auto-Optimization Loop

Automated improvement cycle for PinchTab benchmarks.

## Pre-requisite (Human, once)

```bash
cd ~/dev/pinchtab/tests/benchmark
docker compose up -d --build
# Wait for healthy:
curl -sf -H "Authorization: Bearer benchmark-token" http://localhost:9867/health
```

## Loop Steps

```
┌─────────────────────────────────────────────────────────────┐
│  1a. RUN BASELINE BENCHMARK              [Sub-agent A]      │
│      - Only runs when:                                      │
│        • No previous baseline exists, OR                    │
│        • Changes affect: server code, fixtures, or          │
│          BENCHMARK_TASKS.md                                 │
│      - Otherwise: reuse last baseline results               │
│      - Load BENCHMARK_TASKS.md                              │
│      - Execute 68 curl commands sequentially                │
│      - Check pass/fail per documented conditions            │
│      - Record via: record-step.sh --type baseline           │
│      - Output: results/baseline_<timestamp>.json            │
├─────────────────────────────────────────────────────────────┤
│  1b. RUN AGENT BENCHMARK                [Sub-agent B]      │
│      - Runs every iteration                                 │
│      - Load AGENT_TASKS.md + skills/pinchtab/SKILL.md       │
│      - Execute 39 tasks using only skill docs               │
│      - Record via: record-step.sh --type agent --tokens     │
│      - Output: results/agent_benchmark_<timestamp>.json     │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  2. CHECK FOR REGRESSION                 [Main agent]       │
│     - Read both JSON reports for pass/fail counts           │
│     - Compare against results/best_score.txt                │
│     - If worse: flag regression, prioritize fix/revert      │
│     - If equal or better: update best_score.txt             │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  3. ANALYZE DIFFERENCES                  [Main agent]       │
│     - Diff baseline vs agent results side by side           │
│     - Identify which steps agent failed but baseline passed │
│     - Find patterns (wrong endpoint, selectors, etc.)       │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  4. PROPOSE 1 IMPROVEMENT                [Main agent → Human]│
│     Priority order:                                         │
│     a) Fix PinchTab CLI/REST bug if found                   │
│     b) Improve skill documentation if agent confused        │
│     c) Add verification to existing test case               │
│     d) Add new test case for uncovered scenario             │
│     Present proposal to human for approval.                 │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  5. IMPLEMENT (no commit)                [Main agent]       │
│     - Make the single change                                │
│     - Leave changes uncommitted for manual review           │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│  6. LOG RUN                              [Main agent]       │
│     - Append to results/optimization_log.md                 │
│     - Record: timestamp, pass rates, regression check,      │
│       change made, next focus                               │
└─────────────────────────────────────────────────────────────┘
```

## Roles

| Step | Who | What |
|------|-----|------|
| Pre-req | **Human** | Start Docker, verify health |
| 1a Baseline | **Sub-agent A** | Execute 68 curl commands (skip if baseline exists and no server/fixture changes) |
| 1b Agent | **Sub-agent B** | Execute 39 tasks from skill docs, record results |
| 2 Regression | **Main agent** | Compare against best_score.txt |
| 3 Analyze | **Main agent** | Diff baseline vs agent, find patterns |
| 4 Propose | **Main agent → Human** | Suggest 1 fix, get approval |
| 5 Implement | **Main agent** | Make the change, leave uncommitted |
| 6 Log | **Main agent** | Append to optimization_log.md |

## Regression Detection

Each run compares against `results/best_score.txt`, which tracks the highest pass rates achieved:

```
baseline=62/68
agent=33/39
```

- **If current run is worse**: Regression detected. The improvement priority shifts to fixing or reverting the last change before proposing anything new.
- **If equal or better**: Update `best_score.txt` with the new high-water mark.

## Improvement Priority

1. **API/CLI Bug** — If a curl command returns unexpected error, investigate PinchTab code
2. **Skill Gap** — If agent uses wrong endpoint/approach, improve SKILL.md documentation
3. **Benchmark Gap** — If tests don't verify important behavior, add verification
4. **Coverage Gap** — If common scenarios aren't tested, add test cases

## Log Format

Each run appends to `results/optimization_log.md`:

```markdown
## Run #N — YYYY-MM-DD HH:MM

**Results:**
- Baseline: X/Y (Z%)
- Agent: X/Y (Z%)

**Regression Check:**
- Best: baseline X/Y, agent X/Y
- This run: [better | same | REGRESSION]

**Analysis:**
- [What differed between runs]
- [Root cause identified]

**Change Made:**
- [Type: api|skill|benchmark]
- [Description]
- [Status: uncommitted — pending manual review]

**Next Focus:**
- [What to look at next run]
```

## Files

- `run-optimization.sh` — Main loop script
- `results/optimization_log.md` — Run history
- `results/best_score.txt` — High-water mark for regression detection
- Changes left uncommitted for manual review and commit
