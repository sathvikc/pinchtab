# E2E Test Suite

End-to-end tests for PinchTab that exercise the full stack including browser automation.

## Quick Start

### With Docker (recommended)

```bash
./dev e2e          # Run the extended suite quietly by default
./dev e2e basic    # Run the basic suite (api + cli + infra basic tests)
./dev e2e extended # Run the extended suite
./dev e2e smoke    # Run smoke scenarios plus host Docker smoke checks
./dev e2e smoke-docker # Run host Docker smoke checks only
./dev e2e api      # Run API basic tests
./dev e2e cli      # Run CLI basic tests
./dev e2e infra    # Run infra basic tests
./dev e2e api-extended   # Run API extended tests
./dev e2e cli-extended   # Run CLI extended tests
./dev e2e infra-extended # Run infra extended tests
./dev e2e api logs=show  # Opt back into full streaming logs
```

Or directly through the Go runner:
```bash
go run ./tests/tools/runner e2e --suite basic
go run ./tests/tools/runner e2e --suite extended
go run ./tests/tools/runner e2e --suite smoke
go run ./tests/tools/runner e2e --suite smoke-docker
go run ./tests/tools/runner e2e --suite infra-extended --filter orchestrator
```

## Architecture

```
tests/e2e/
├── docker-compose.yml      # Single-instance stack for basic suites
├── docker-compose-multi.yml # Multi-instance extended stack
├── config/                 # E2E-specific PinchTab configs
│   ├── pinchtab.json
│   ├── pinchtab-medium-permissive.json
│   ├── pinchtab-full-permissive.json
│   ├── pinchtab-secure.json
│   └── pinchtab-bridge.json
├── fixtures/               # Static HTML test pages
│   ├── index.html
│   ├── form.html
│   ├── table.html
│   └── buttons.html
├── helpers/                # Shared API/CLI E2E helpers
│   ├── api.sh
│   ├── api-http.sh
│   ├── api-assertions.sh
│   ├── api-actions.sh
│   ├── api-snapshot.sh
│   ├── cli.sh
│   └── base.sh
├── scenarios/              # Test scenarios organized by type
│   ├── api/                # Browser control and page interaction
│   │   ├── browser-basic.sh
│   │   ├── browser-extended.sh
│   │   ├── tabs-basic.sh
│   │   ├── tabs-extended.sh
│   │   ├── actions-basic.sh
│   │   ├── actions-extended.sh
│   │   ├── files-basic.sh
│   │   ├── files-extended.sh
│   │   ├── clipboard-basic.sh
│   │   └── console-basic.sh
│   ├── cli/                # CLI command tests
│   │   ├── browser-basic.sh
│   │   ├── browser-extended.sh
│   │   ├── tabs-basic.sh
│   │   ├── tabs-extended.sh
│   │   └── ...
│   └── infra/              # System, network, security, stealth
│       ├── system-basic.sh
│       ├── system-extended.sh
│       ├── network-basic.sh
│       ├── network-extended.sh
│       ├── security-basic.sh
│       ├── security-extended.sh
│       ├── stealth-basic.sh
│       ├── stealth-extended.sh
│       ├── orchestrator-extended.sh
│       ├── auth-extended.sh
│       └── ...
├── runner-api/             # API test runner container
│   └── Dockerfile
├── runner-cli/             # CLI test runner container
│   └── Dockerfile
└── results/                # Test output (gitignored)
```

The Docker stacks reuse the repository root `Dockerfile` and mount explicit config files with `PINCHTAB_CONFIG` instead of maintaining separate e2e-only images.

## Test Groups

Tests are organized into three parallel groups:

### API Group (`scenarios/api/`)
Browser control and page interaction tests:
- `browser-basic` / `browser-extended`
- `tabs-basic` / `tabs-extended`
- `actions-basic` / `actions-extended`
- `files-basic` / `files-extended`
- `clipboard-basic`
- `console-basic`

### CLI Group (`scenarios/cli/`)
CLI command tests:
- `browser-basic` / `browser-extended`
- `tabs-basic` / `tabs-extended`
- `actions-basic` / `actions-extended`
- `files-basic` / `files-extended`
- `system-basic` / `system-extended`
- And more...

### Infra Group (`scenarios/infra/`)
System, networking, security, and stealth tests:
- `system-basic` / `system-extended`
- `network-basic` / `network-extended`
- `security-basic` / `security-extended`
- `stealth-basic` / `stealth-extended`
- `orchestrator-extended`
- `auth-extended`
- `autosolver-smoke`
- `dashboard-smoke`
- manual autosolver check lives at `tests/manual/autosolver-check.sh`
- real-world autosolver smoke lives at `scripts/autosolver-realworld-smoke.sh`
- `idpi-extended`

The `basic` entrypoints are the PR happy path. The `extended` entrypoints add extra and edge-case coverage; extended suites include their matching basic scenarios. The `smoke` tier is separate and runs only `*-smoke.sh` scenarios plus host-level Docker smoke checks, not basic or extended scenarios. The Go runner selects the exact scenario files before entering the container; `run.sh` is only the in-container executor and requires explicit `scenario=<file>` arguments from the host runner.

Compose usage:
- `docker-compose.yml` powers `api`, `cli`, `infra`, and `cli-extended`
- `docker-compose-multi.yml` powers `api-extended` and `infra-extended`

## Adding Tests

1. Add or update a grouped entrypoint such as `tabs-basic.sh` or `tabs-extended.sh`
2. Source `../../helpers/api.sh` or `../../helpers/cli.sh`
3. Put the happy path in `*-basic.sh` and the extra/edge cases in `*-extended.sh`
4. Use the assertion helpers:

```bash
#!/bin/bash
GROUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${GROUP_DIR}/../../helpers/api.sh"

start_test "My test name"

# Assert HTTP status
assert_status 200 "${PINCHTAB_URL}/health"

# Assert JSON field equals value
RESULT=$(pt_get "/some/endpoint")
assert_json_eq "$RESULT" '.field' 'expected'

# Assert JSON contains substring
assert_json_contains "$RESULT" '.message' 'success'

# Assert array length
assert_json_length "$RESULT" '.items' 5

end_test
```

The action scenarios already cover common interaction regressions against the bundled fixtures:
- `tests/e2e/scenarios/api/actions-basic.sh` groups the API happy-path actions
- `tests/e2e/scenarios/cli/actions-basic.sh` groups the matching CLI commands

## Adding Fixtures

Add HTML files to `fixtures/` for testing specific scenarios:

- Forms and inputs
- Tables and data
- Dynamic content
- iframes
- File upload/download

## CI Integration

The E2E tests run automatically:
- On PRs: `api`, `cli`, and `infra` basic tests always run
- On PRs: touching any non-basic scenario also triggers the matching extended suite on its native compose stack
- On PRs: touching smoke scenarios or Docker smoke inputs triggers the smoke workflow
- Manually via workflow dispatch: Extended tests for all groups

## Result Files

The Go e2e runner captures runner output and writes each suite's result files in `tests/e2e/results/`. The container executor emits structured `E2E_RESULT` lines as tests finish, and the Go layer turns those into the final console summary, GitHub Actions summaries, durable summaries, and reports:

- `summary-api.txt` / `report-api.md`
- `summary-api-extended.txt` / `report-api-extended.md`
- `summary-cli.txt` / `report-cli.md`
- `summary-cli-extended.txt` / `report-cli-extended.md`
- `summary-infra.txt` / `report-infra.md`
- `summary-infra-extended.txt` / `report-infra-extended.md`
- `summary-api-smoke.txt` / `report-api-smoke.md`
- `summary-cli-smoke.txt` / `report-cli-smoke.md`
- `summary-infra-smoke.txt` / `report-infra-smoke.md`
- `summary-plugin-smoke.txt` / `report-plugin-smoke.md`
- `summary-docker-smoke.txt` / `report-docker-smoke.md`

The runner deletes the target suite files before each run to avoid stale output.
Saved and printed summaries include total test time and suite wall time.

The runner also saves the captured suite log under `output-*.log` and captures compose service logs on failure. When `logs=hide` is used and a suite fails, it prints the relevant failure summary and artifact paths.

## Debugging

### View container logs
```bash
docker compose -f tests/e2e/docker-compose.yml logs pinchtab
docker compose -f tests/e2e/docker-compose-multi.yml logs pinchtab
```

### Interactive shell in runner
```bash
docker compose -f tests/e2e/docker-compose.yml run runner-api bash
docker compose -f tests/e2e/docker-compose.yml run runner-cli bash
```

### Run specific scenario
```bash
docker compose -f tests/e2e/docker-compose.yml run runner-api /bin/bash /e2e/scenarios/api/tabs-basic.sh
docker compose -f tests/e2e/docker-compose-multi.yml run runner-api /bin/bash /e2e/scenarios/api/tabs-extended.sh
```

### Orchestrator Coverage
`infra-extended` uses `docker-compose-multi.yml` and includes the multi-instance and remote-bridge orchestrator scenarios through `orchestrator-extended.sh`.
