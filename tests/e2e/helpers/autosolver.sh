#!/bin/bash
# autosolver-helper.sh — RunAutoSolverAndExtract helper for autosolver E2E tests.
#
# Provides:
#   run_autosolver_and_extract <label> <fixture_path> <js_expr> [sleep_sec]
#
# Steps:
#   1. Navigate to the fixture URL
#   2. Wait for JS to execute (configurable sleep)
#   3. Evaluate js_expr to extract page signals
#   4. Print result JSON to stdout / test log
#   5. On failure, dump page text for debugging
#
# Usage:
#   source helpers/autosolver-helper.sh
#   AUTOSOLVER_RESULT=$(run_autosolver_and_extract "bot-detect" "bot-detect.html" \
#     "JSON.stringify(window.__botDetectScore || null)")

HELPERS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${HELPERS_DIR}/base.sh"
source "${HELPERS_DIR}/api-assertions.sh"
source "${HELPERS_DIR}/api-http.sh"

# ─────────────────────────────────────────────────────────────────────────────
# run_autosolver_and_extract
#   $1  label       — human-readable name for log output
#   $2  fixture     — path relative to FIXTURES_URL (e.g. "bot-detect.html")
#   $3  js_expr     — JavaScript expression to evaluate after page load
#   $4  sleep_sec   — (optional) seconds to wait after navigate; default 1
#
# Sets global AUTOSOLVER_RESULT with the evaluated JSON string.
# Returns 0 if navigate succeeded, 1 otherwise.
# ─────────────────────────────────────────────────────────────────────────────
AUTOSOLVER_RESULT=""
AUTOSOLVER_PAGE_TEXT=""

run_autosolver_and_extract() {
  local label="$1"
  local fixture="$2"
  local js_expr="$3"
  local sleep_sec="${4:-1}"

  echo -e "${BLUE}[autosolver] navigating to ${fixture}${NC}"
  pt_post /navigate "{\"url\":\"${FIXTURES_URL}/${fixture}\"}"
  if [ "$HTTP_STATUS" != "200" ] && [ "$HTTP_STATUS" != "201" ]; then
    echo -e "  ${RED}✗${NC} [$label] navigate failed (HTTP $HTTP_STATUS)"
    return 1
  fi
  echo -e "  ${GREEN}✓${NC} [$label] navigated (tab: $(echo "$RESULT" | jq -r '.tabId // "?"'))"

  sleep "$sleep_sec"

  # Extract the requested JS value.
  local eval_body
  eval_body=$(jq -n --arg expr "$js_expr" '{"expression": $expr}')
  pt_post /evaluate "$eval_body"
  AUTOSOLVER_RESULT=$(echo "$RESULT" | jq -r '.result // "null"')

  echo -e "  ${MUTED}[$label] extracted: ${AUTOSOLVER_RESULT:0:200}${NC}"

  # Also capture page text for debugging on failure.
  local text_body
  text_body=$(jq -n '{"expression": "document.body ? document.body.innerText.substring(0,2000) : \"\""}')
  pt_post /evaluate "$text_body"
  AUTOSOLVER_PAGE_TEXT=$(echo "$RESULT" | jq -r '.result // ""')

  return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# dump_autosolver_debug
#   Prints full diagnostic info when a test fails.
#   Call after an assertion block when debugging is needed.
# ─────────────────────────────────────────────────────────────────────────────
dump_autosolver_debug() {
  local label="${1:-autosolver}"
  echo ""
  echo -e "${YELLOW}════ AUTOSOLVER DEBUG: ${label} ════${NC}"
  echo -e "${MUTED}Extracted result:${NC}"
  echo "  $AUTOSOLVER_RESULT" | jq '.' 2>/dev/null || echo "  $AUTOSOLVER_RESULT"
  echo ""
  echo -e "${MUTED}Page text (first 1000 chars):${NC}"
  echo "$AUTOSOLVER_PAGE_TEXT" | head -c 1000
  echo ""
  echo -e "${YELLOW}══════════════════════════════════${NC}"
  echo ""
}

# ─────────────────────────────────────────────────────────────────────────────
# assert_autosolver_score
#   Parses a __botDetectScore / __cdpDetectScore JSON and asserts:
#     - passed == true
#     - critical >= min_critical
#   $1  score_json  — JSON string from window.__botDetectScore etc.
#   $2  label       — description for log
#   $3  min_critical — minimum critical tests that must pass (default: all)
# ─────────────────────────────────────────────────────────────────────────────
assert_autosolver_score() {
  local score_json="$1"
  local label="$2"
  local min_critical="${3:-}"

  local passed critical critical_total
  passed=$(echo "$score_json" | jq -r '.passed // false')
  critical=$(echo "$score_json" | jq -r '.critical // 0')
  critical_total=$(echo "$score_json" | jq -r '.criticalTotal // 0')

  echo -e "  ${MUTED}[$label] score: critical=${critical}/${critical_total} passed=${passed}${NC}"

  if [ -z "$min_critical" ]; then
    min_critical="$critical_total"
  fi

  if [ "$passed" = "true" ] && [ "$critical" -ge "$min_critical" ]; then
    echo -e "  ${GREEN}✓${NC} [$label] score: passed (critical ${critical}/${critical_total})"
    ((ASSERTIONS_PASSED++)) || true
  else
    echo -e "  ${RED}✗${NC} [$label] score: failed (critical ${critical}/${critical_total}, need ${min_critical})"
    ((ASSERTIONS_FAILED++)) || true
    dump_autosolver_debug "$label"
  fi
}
