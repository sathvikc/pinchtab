#!/bin/bash
# run-recent.sh - Run only recently added/changed E2E test scenarios
# This runs a fast subset for fail-fast CI before the full suites.

set -uo pipefail

SCRIPT_DIR="$(dirname "$0")"
source "${SCRIPT_DIR}/common.sh"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}PinchTab E2E Recent Tests${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "PINCHTAB_URL: ${PINCHTAB_URL}"
echo "FIXTURES_URL: ${FIXTURES_URL}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo "Waiting for instances to become ready..."
wait_for_instance_ready "${PINCHTAB_URL}"
wait_for_instance_ready "${PINCHTAB_SECURE_URL}"
echo ""

# Recent test files — add new scenarios here for fast CI feedback.
# Move to the full suite (run-all.sh) once stable.
RECENT_TESTS=(
  "41-extensions.sh"
)

for name in "${RECENT_TESTS[@]}"; do
  script="${SCRIPT_DIR}/${name}"
  if [ -f "$script" ]; then
    echo -e "${YELLOW}Running: ${name}${NC}"
    echo ""
    source "$script"
    echo ""
  else
    echo -e "${RED}Missing: ${name}${NC}"
  fi
done

print_summary

if [ -d "${RESULTS_DIR:-}" ]; then
  echo "passed=$TESTS_PASSED" > "${RESULTS_DIR}/summary.txt"
  echo "failed=$TESTS_FAILED" >> "${RESULTS_DIR}/summary.txt"
  echo "timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "${RESULTS_DIR}/summary.txt"
fi
