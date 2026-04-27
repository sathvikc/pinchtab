#!/usr/bin/env bash
# dev-e2e.sh — run a single E2E test by name, with a fresh build.
#
# Usage:
#   scripts/dev-e2e.sh "<test name substring>"
#   scripts/dev-e2e.sh "humanClick: click input by ref"
#   scripts/dev-e2e.sh "scroll (down)"
#
# Locates the matching scenario file, infers the suite (api/cli/infra/plugin)
# and whether it's an -extended scenario, then dispatches the Go e2e runner
# with the right filter+test arguments.

set -euo pipefail

cd "$(dirname "$0")/.."

if [ "$#" -lt 1 ] || [ -z "$1" ]; then
  echo "usage: $0 \"<test name substring>\"" >&2
  echo "       Locate a single E2E test by its start_test name and run only that one." >&2
  exit 2
fi

TEST_NAME="$1"

# Match `start_test "..."` lines whose quoted name contains the requested
# substring. fgrep keeps the user's input literal (no regex surprises).
matches=$(grep -rn -F -- "start_test" tests/e2e/scenarios \
  | grep -F -- "${TEST_NAME}" \
  | grep -E '^[^:]+:[0-9]+:[[:space:]]*start_test[[:space:]]+' || true)

if [ -z "${matches}" ]; then
  echo "no test name matches \"${TEST_NAME}\" in tests/e2e/scenarios" >&2
  exit 1
fi

# Use the first match; warn if there are several so the caller can disambiguate.
match_count=$(printf '%s\n' "${matches}" | wc -l | tr -d ' ')
if [ "${match_count}" -gt 1 ]; then
  echo "multiple tests matched \"${TEST_NAME}\":" >&2
  printf '%s\n' "${matches}" | sed 's/^/  /' >&2
  echo "" >&2
  echo "using the first one. Pass a longer/more-specific substring to pick a different test." >&2
  echo "" >&2
fi

scenario_path=$(printf '%s\n' "${matches}" | head -n1 | cut -d: -f1)
scenario_file=$(basename "${scenario_path}")
scenario_dir=$(basename "$(dirname "${scenario_path}")")
scenario_stem="${scenario_file%.sh}"

# Map scenario dir to the Go runner suite.
case "${scenario_dir}" in
  api|cli|infra|plugin) suite="${scenario_dir}" ;;
  *)
    echo "unrecognized scenario directory: ${scenario_dir}" >&2
    exit 1
    ;;
esac

# Plugin suite has no extended variant; basic/extended scenarios in api/cli/
# infra map to the corresponding -extended dispatcher entry.
if [ "${suite}" != "plugin" ] && [[ "${scenario_file}" == *-extended.sh ]]; then
  dispatch="${suite}-extended"
else
  dispatch="${suite}"
fi

echo "▶ test:     ${TEST_NAME}"
echo "  scenario: ${scenario_path}"
echo "  suite:    ${dispatch}"
echo ""

exec env E2E_LOGS="${E2E_LOGS:-show}" go run ./tests/tools/runner e2e \
  --suite "${dispatch}" \
  --filter "${scenario_stem}" \
  --test "${TEST_NAME}"
