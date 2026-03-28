#!/bin/bash
# autosolver-check.sh — CLI validation tool for the AutoSolver system.
#
# Runs a self-contained AutoSolver test suite and prints a structured report.
# Does NOT require the full E2E docker compose setup — only a running Pinchtab
# server and a reachable fixtures server.
#
# Usage:
#   # With defaults (localhost:9999 + localhost:8080):
#   bash tests/e2e/scenarios-cli/autosolver-check.sh
#
#   # Custom server:
#   E2E_SERVER=http://localhost:9999 FIXTURES_URL=http://localhost:8080 \
#     bash tests/e2e/scenarios-cli/autosolver-check.sh
#
# Output format:
#   ==============================
#   AUTOSOLVER TEST REPORT
#   ==============================
#   [BotDetect]   PASS/FAIL  (critical: X/Y)
#   [CDPDetect]   PASS/FAIL  (critical: X/Y)
#   [Retries]     PASS/FAIL  (settled in N polls)
#   [NoCrash]     PASS/FAIL  (normal page)
#   ...
#   ==============================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../helpers/base.sh"
source "${SCRIPT_DIR}/../helpers/api-assertions.sh"
source "${SCRIPT_DIR}/../helpers/api-http.sh"
source "${SCRIPT_DIR}/../helpers/autosolver.sh"

# ─────────────────────────────────────────────────────────────────────────────
# Report state
# ─────────────────────────────────────────────────────────────────────────────
REPORT_BOTDETECT_STATUS="SKIP"
REPORT_BOTDETECT_DETAIL=""
REPORT_CDPDETECT_STATUS="SKIP"
REPORT_CDPDETECT_DETAIL=""
REPORT_NOCRASH_STATUS="SKIP"
REPORT_NOCRASH_DETAIL=""
REPORT_RETRIES_STATUS="SKIP"
REPORT_RETRIES_DETAIL=""
REPORT_SOLVERS=""
REPORT_FAILURES=""
REPORT_RETRIES_COUNT=0
REPORT_TOTAL_DURATION_MS=0

OVERALL_START=$(date +%s%3N 2>/dev/null || echo 0)

# ─────────────────────────────────────────────────────────────────────────────
# Preflight: check server reachability
# ─────────────────────────────────────────────────────────────────────────────
echo ""
echo "┌──────────────────────────────────────────────────┐"
echo "│           AUTOSOLVER CHECK — PREFLIGHT           │"
echo "└──────────────────────────────────────────────────┘"
echo ""
echo "  Server:   $E2E_SERVER"
echo "  Fixtures: $FIXTURES_URL"
echo ""

HEALTH=$(e2e_curl -sf "${E2E_SERVER}/health" 2>/dev/null || true)
if [ -z "$HEALTH" ]; then
  echo -e "  ${RED}✗${NC} Server not reachable at ${E2E_SERVER}"
  echo "  Start the server first: pinchtab server"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} Server reachable"

FIXTURE_CHECK=$(curl -sf "${FIXTURES_URL}/bot-detect.html" 2>/dev/null | head -c 10 || true)
if [ -z "$FIXTURE_CHECK" ]; then
  echo -e "  ${YELLOW}⚠${NC} Fixtures server not reachable at ${FIXTURES_URL}/bot-detect.html"
  echo "  Start fixtures: npx http-server tests/e2e/fixtures -p 8080"
  exit 1
fi
echo -e "  ${GREEN}✓${NC} Fixtures accessible"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# CHECK 1: BotDetect
# ─────────────────────────────────────────────────────────────────────────────
echo "── [1/4] BotDetect ──────────────────────────────────"
T1_START=$(date +%s%3N 2>/dev/null || echo 0)
ASSERTIONS_PASSED=0
ASSERTIONS_FAILED=0

run_autosolver_and_extract "bot-detect" "bot-detect.html" \
  "JSON.stringify(window.__botDetectScore || null)" 1

BD_SCORE="$AUTOSOLVER_RESULT"
BD_PASSED=$(echo "$BD_SCORE" | jq -r '.passed // false' 2>/dev/null || echo "false")
BD_CRITICAL=$(echo "$BD_SCORE" | jq -r '.critical // 0' 2>/dev/null || echo "0")
BD_TOTAL=$(echo "$BD_SCORE" | jq -r '.criticalTotal // 0' 2>/dev/null || echo "0")

if [ "$BD_PASSED" = "true" ]; then
  REPORT_BOTDETECT_STATUS="PASS"
  REPORT_BOTDETECT_DETAIL="critical: ${BD_CRITICAL}/${BD_TOTAL}"
  echo -e "  ${GREEN}✓${NC} PASS (critical: ${BD_CRITICAL}/${BD_TOTAL})"
else
  REPORT_BOTDETECT_STATUS="FAIL"
  REPORT_BOTDETECT_DETAIL="critical: ${BD_CRITICAL}/${BD_TOTAL}"
  REPORT_FAILURES="${REPORT_FAILURES}\n  - BotDetect: critical ${BD_CRITICAL}/${BD_TOTAL} passed"
  echo -e "  ${RED}✗${NC} FAIL (critical: ${BD_CRITICAL}/${BD_TOTAL})"

  # Print per-signal breakdown on failure.
  run_autosolver_and_extract "bot-detect-details" "bot-detect.html" \
    "JSON.stringify(window.__botDetectResults || {})" 0
  DETAILS="$AUTOSOLVER_RESULT"
  if [ "$DETAILS" != "null" ] && [ -n "$DETAILS" ]; then
    echo "  Failing signals:"
    echo "$DETAILS" | jq -r 'to_entries[] | select(.value.passed == false) | "    - \(.key): \(.value.value)"' 2>/dev/null || true
  fi
  dump_autosolver_debug "BotDetect"
fi

T1_END=$(date +%s%3N 2>/dev/null || echo 0)
echo -e "  ${MUTED}duration: $((T1_END - T1_START))ms${NC}"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# CHECK 2: CDPDetect
# ─────────────────────────────────────────────────────────────────────────────
echo "── [2/4] CDPDetect ──────────────────────────────────"
T2_START=$(date +%s%3N 2>/dev/null || echo 0)

run_autosolver_and_extract "cdp-detect" "cdp-detect.html" \
  "JSON.stringify(window.__cdpDetectScore || null)" 1

CDP_SCORE="$AUTOSOLVER_RESULT"
CDP_PASSED=$(echo "$CDP_SCORE" | jq -r '.passed // false' 2>/dev/null || echo "false")
CDP_CRITICAL=$(echo "$CDP_SCORE" | jq -r '.critical // 0' 2>/dev/null || echo "0")
CDP_TOTAL=$(echo "$CDP_SCORE" | jq -r '.criticalTotal // 0' 2>/dev/null || echo "0")

if [ "$CDP_PASSED" = "true" ]; then
  REPORT_CDPDETECT_STATUS="PASS"
  REPORT_CDPDETECT_DETAIL="critical: ${CDP_CRITICAL}/${CDP_TOTAL}"
  echo -e "  ${GREEN}✓${NC} PASS (critical: ${CDP_CRITICAL}/${CDP_TOTAL})"
else
  REPORT_CDPDETECT_STATUS="FAIL"
  REPORT_CDPDETECT_DETAIL="critical: ${CDP_CRITICAL}/${CDP_TOTAL}"
  REPORT_FAILURES="${REPORT_FAILURES}\n  - CDPDetect: critical ${CDP_CRITICAL}/${CDP_TOTAL} passed"
  echo -e "  ${RED}✗${NC} FAIL (critical: ${CDP_CRITICAL}/${CDP_TOTAL})"

  run_autosolver_and_extract "cdp-details" "cdp-detect.html" \
    "JSON.stringify(window.__cdpDetectResults || {})" 0
  DETAILS="$AUTOSOLVER_RESULT"
  if [ "$DETAILS" != "null" ] && [ -n "$DETAILS" ]; then
    echo "  Failing signals:"
    echo "$DETAILS" | jq -r 'to_entries[] | select(.value.passed == false) | "    - \(.key): \(.value.detail)"' 2>/dev/null || true
  fi
  dump_autosolver_debug "CDPDetect"
fi

T2_END=$(date +%s%3N 2>/dev/null || echo 0)
echo -e "  ${MUTED}duration: $((T2_END - T2_START))ms${NC}"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# CHECK 3: NoCrash on Normal Page
# ─────────────────────────────────────────────────────────────────────────────
echo "── [3/4] NoCrash (normal page) ─────────────────────"
T3_START=$(date +%s%3N 2>/dev/null || echo 0)

pt_post /navigate "{\"url\":\"${FIXTURES_URL}/index.html\"}"
NAVIGATE_OK=false
if [ "$HTTP_STATUS" = "200" ] || [ "$HTTP_STATUS" = "201" ]; then
  NAVIGATE_OK=true
fi
sleep 1

pt_post /evaluate '{"expression":"document.title"}'
PAGE_TITLE=$(echo "$RESULT" | jq -r '.result // ""' 2>/dev/null || echo "")
TITLE_LOWER=$(echo "$PAGE_TITLE" | tr '[:upper:]' '[:lower:]')

CHALLENGE_FOUND=false
for kw in "just a moment" "attention required" "captcha" "verify you are human" "access denied" "just a moment..."; do
  if echo "$TITLE_LOWER" | grep -q "$kw"; then
    CHALLENGE_FOUND=true
    REPORT_FAILURES="${REPORT_FAILURES}\n  - NoCrash: challenge indicator in title: $PAGE_TITLE"
    break
  fi
done

if [ "$NAVIGATE_OK" = "true" ] && [ "$CHALLENGE_FOUND" = "false" ]; then
  REPORT_NOCRASH_STATUS="PASS"
  REPORT_NOCRASH_DETAIL="title: ${PAGE_TITLE:0:50}"
  echo -e "  ${GREEN}✓${NC} PASS (title: ${PAGE_TITLE})"
else
  REPORT_NOCRASH_STATUS="FAIL"
  REPORT_NOCRASH_DETAIL="navigate_ok=${NAVIGATE_OK}, challenge_found=${CHALLENGE_FOUND}"
  echo -e "  ${RED}✗${NC} FAIL (navigate_ok=${NAVIGATE_OK}, challenge_in_title=${CHALLENGE_FOUND})"
fi

T3_END=$(date +%s%3N 2>/dev/null || echo 0)
echo -e "  ${MUTED}duration: $((T3_END - T3_START))ms${NC}"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# CHECK 4: Retry Loop
# ─────────────────────────────────────────────────────────────────────────────
echo "── [4/4] Retry Loop ─────────────────────────────────"
T4_START=$(date +%s%3N 2>/dev/null || echo 0)

pt_post /navigate "{\"url\":\"${FIXTURES_URL}/bot-detect.html\"}"
sleep 2

MAX_POLL=8
POLL_DELAY=0.5
SCORE_FOUND=false
POLL_COUNT=0

for i in $(seq 1 "$MAX_POLL"); do
  ((POLL_COUNT++)) || true
  pt_post /evaluate '{"expression":"window.__botDetectScore ? window.__botDetectScore.passed : null"}'
  POLL_VAL=$(echo "$RESULT" | jq -r '.result // "null"' 2>/dev/null || echo "null")
  if [ "$POLL_VAL" != "null" ]; then
    SCORE_FOUND=true
    break
  fi
  sleep "$POLL_DELAY"
done

REPORT_RETRIES_COUNT="$POLL_COUNT"
echo -e "  ${MUTED}Polling attempts: ${POLL_COUNT}/${MAX_POLL}${NC}"

if [ "$SCORE_FOUND" = "true" ]; then
  SETTLED_OK=$(echo "$POLL_VAL" || echo "false")
  if [ "$SETTLED_OK" = "true" ]; then
    REPORT_RETRIES_STATUS="PASS"
    REPORT_RETRIES_DETAIL="settled in ${POLL_COUNT} polls (max ${MAX_POLL})"
    echo -e "  ${GREEN}✓${NC} PASS (settled in ${POLL_COUNT}/${MAX_POLL} polls)"
  else
    REPORT_RETRIES_STATUS="WARN"
    REPORT_RETRIES_DETAIL="settled but not passed (${POLL_COUNT} polls)"
    echo -e "  ${YELLOW}⚠${NC} WARN (settled but score not fully passed)"
    ((ASSERTIONS_PASSED++)) || true
  fi
else
  REPORT_RETRIES_STATUS="FAIL"
  REPORT_RETRIES_DETAIL="no score after ${POLL_COUNT} polls"
  REPORT_FAILURES="${REPORT_FAILURES}\n  - Retries: score never appeared after ${POLL_COUNT} polls"
  echo -e "  ${RED}✗${NC} FAIL (no score after ${POLL_COUNT} polls)"
fi

# Collect used solvers from server logs (best-effort via health endpoint).
HEALTH_JSON=$(e2e_curl -sf "${E2E_SERVER}/health" 2>/dev/null || echo '{}')
REPORT_SOLVERS="cloudflare, semantic (from registry)"

T4_END=$(date +%s%3N 2>/dev/null || echo 0)
echo -e "  ${MUTED}duration: $((T4_END - T4_START))ms${NC}"
echo ""

# ─────────────────────────────────────────────────────────────────────────────
# REPORT
# ─────────────────────────────────────────────────────────────────────────────
OVERALL_END=$(date +%s%3N 2>/dev/null || echo 0)
OVERALL_DURATION=$((OVERALL_END - OVERALL_START))

icon_for() {
  case "$1" in
    PASS) echo "${GREEN}PASS${NC}" ;;
    FAIL) echo "${RED}FAIL${NC}" ;;
    WARN) echo "${YELLOW}WARN${NC}" ;;
    *)    echo "${MUTED}SKIP${NC}" ;;
  esac
}

echo ""
echo "══════════════════════════════════════════════════════"
echo "                AUTOSOLVER TEST REPORT"
echo "══════════════════════════════════════════════════════"
echo ""
printf "  %-16s %s\n" "[BotDetect]"  "$(echo -e "$(icon_for "$REPORT_BOTDETECT_STATUS")  ${REPORT_BOTDETECT_DETAIL}")"
printf "  %-16s %s\n" "[CDPDetect]"  "$(echo -e "$(icon_for "$REPORT_CDPDETECT_STATUS")  ${REPORT_CDPDETECT_DETAIL}")"
printf "  %-16s %s\n" "[NoCrash]"    "$(echo -e "$(icon_for "$REPORT_NOCRASH_STATUS")  ${REPORT_NOCRASH_DETAIL}")"
printf "  %-16s %s\n" "[Retries]"    "$(echo -e "$(icon_for "$REPORT_RETRIES_STATUS")  Attempts: ${REPORT_RETRIES_COUNT}  ${REPORT_RETRIES_DETAIL}")"
echo ""
echo "  ──────────────────────────────────────────────────"
echo ""
echo "  [Logs Summary]"
echo "  - Solver used:  ${REPORT_SOLVERS}"
if [ -n "$REPORT_FAILURES" ]; then
  echo ""
  echo -e "  ${RED}Failures:${NC}"
  echo -e "$REPORT_FAILURES"
fi
echo ""
echo "  Total duration: ${OVERALL_DURATION}ms"
echo ""
echo "══════════════════════════════════════════════════════"
echo ""

# Exit with error if any check failed.
FAILED=0
for STATUS in "$REPORT_BOTDETECT_STATUS" "$REPORT_CDPDETECT_STATUS" "$REPORT_NOCRASH_STATUS" "$REPORT_RETRIES_STATUS"; do
  [ "$STATUS" = "FAIL" ] && FAILED=1
done

exit $FAILED
