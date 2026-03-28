#!/bin/bash
# autosolver-realworld.sh — Real-site AutoSolver simulation for stealth leakage checks.
#
# This scenario is intentionally manual/high-friction and should not be run in CI.
# It compares pre-solve vs post-solve detection signals on real bot-detection pages.
#
# Usage:
#   RUN_REAL_WORLD_AUTOSOLVER=1 \
#   E2E_SERVER=http://localhost:9999 \
#   E2E_SERVER_TOKEN=e2e-token \
#   bash tests/e2e/scenarios-api/autosolver-realworld.sh
#
# Optional:
#   REAL_WORLD_AUTOSOLVER_TARGETS="https://pixelscan.net/bot-check,https://bot.sannysoft.com,https://browserscan.net"
#   REAL_WORLD_NAV_SLEEP_SEC=4
#
# Output:
#   - Per-site pre-solve and post-solve signal table
#   - Delta report (before -> after)
#
# Signals tracked:
#   - Bot Behavior Detected
#   - CDP Detected
#   - NavigatorWebdriver
#   - TamperedFunctions

GROUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${GROUP_DIR}/../helpers/api.sh"

if [ "${RUN_REAL_WORLD_AUTOSOLVER:-0}" != "1" ]; then
  echo "[autosolver-realworld] skipped: set RUN_REAL_WORLD_AUTOSOLVER=1 to run this scenario"
  exit 0
fi

TARGETS_CSV="${REAL_WORLD_AUTOSOLVER_TARGETS:-https://pixelscan.net/bot-check,https://bot.sannysoft.com,https://browserscan.net}"
NAV_SLEEP_SEC="${REAL_WORLD_NAV_SLEEP_SEC:-4}"

split_csv_targets() {
  local csv="$1"
  local old_ifs="$IFS"
  IFS=',' read -r -a TARGETS <<< "$csv"
  IFS="$old_ifs"
}

extract_text_for_tab() {
  local tab_id="$1"
  pt_get "/text?tabId=${tab_id}"
  if [ "$HTTP_STATUS" != "200" ]; then
    echo ""
    return 1
  fi
  echo "$RESULT" | jq -r '.text // ""'
}

contains_signal() {
  local text_lc="$1"
  local regex="$2"
  if printf '%s' "$text_lc" | grep -Eq "$regex"; then
    echo "true"
  else
    echo "false"
  fi
}

capture_signals_json() {
  local raw_text="$1"
  local text_lc
  text_lc=$(printf '%s' "$raw_text" | tr '[:upper:]' '[:lower:]')

  local bot_behavior cdp_detected navigator_webdriver tampered_functions
  bot_behavior=$(contains_signal "$text_lc" "bot behavior detected|bot detected")
  cdp_detected=$(contains_signal "$text_lc" "cdp detected|devtools protocol")
  navigator_webdriver=$(contains_signal "$text_lc" "navigator.?webdriver|webdriver")
  tampered_functions=$(contains_signal "$text_lc" "tampered functions|tamperedfunctions")

  jq -n \
    --argjson bot "$bot_behavior" \
    --argjson cdp "$cdp_detected" \
    --argjson wd "$navigator_webdriver" \
    --argjson tf "$tampered_functions" \
    '{
      botBehaviorDetected: $bot,
      cdpDetected: $cdp,
      navigatorWebdriver: $wd,
      tamperedFunctions: $tf
    }'
}

log_signal_delta() {
  local label="$1"
  local before_json="$2"
  local after_json="$3"

  echo -e "  ${MUTED}${label}${NC}"
  for key in botBehaviorDetected cdpDetected navigatorWebdriver tamperedFunctions; do
    local b a
    b=$(echo "$before_json" | jq -r ".${key}")
    a=$(echo "$after_json" | jq -r ".${key}")
    local marker="="
    if [ "$b" != "$a" ]; then
      marker="->"
    fi
    echo "    ${key}: ${b} ${marker} ${a}"
  done
}

run_site() {
  local url="$1"

  echo -e "${BLUE}[realworld] navigate:${NC} ${url}"
  pt_post /navigate "{\"url\":\"${url}\"}"
  if [ "$HTTP_STATUS" != "200" ] && [ "$HTTP_STATUS" != "201" ]; then
    echo -e "  ${RED}✗${NC} navigate failed (${HTTP_STATUS})"
    if [ "$HTTP_STATUS" = "403" ] && echo "$RESULT" | grep -qi "Domain not in allowlist"; then
      echo -e "  ${YELLOW}⚠${NC} IDPI allowlist is blocking this host"
    fi
    ((ASSERTIONS_FAILED++)) || true
    return
  fi

  local tab_id
  tab_id=$(echo "$RESULT" | jq -r '.tabId // ""')
  if [ -z "$tab_id" ]; then
    echo -e "  ${RED}✗${NC} missing tabId in navigate response"
    ((ASSERTIONS_FAILED++)) || true
    return
  fi

  sleep "$NAV_SLEEP_SEC"

  local before_text
  before_text=$(extract_text_for_tab "$tab_id")
  if [ "$HTTP_STATUS" != "200" ]; then
    echo -e "  ${RED}✗${NC} failed to extract pre-solve text (${HTTP_STATUS})"
    ((ASSERTIONS_FAILED++)) || true
    return
  fi

  local before_signals
  before_signals=$(capture_signals_json "$before_text")

  echo -e "${BLUE}[realworld] solve:${NC} ${url}"
  pt_post /solve "{\"tabId\":\"${tab_id}\",\"maxAttempts\":3,\"timeout\":45000}"
  if [ "$HTTP_STATUS" != "200" ]; then
    echo -e "  ${RED}✗${NC} solve failed (${HTTP_STATUS})"
    ((ASSERTIONS_FAILED++)) || true
    return
  fi

  local solver_name solved attempts challenge_type
  solver_name=$(echo "$RESULT" | jq -r '.solver // ""')
  solved=$(echo "$RESULT" | jq -r '.solved // false')
  attempts=$(echo "$RESULT" | jq -r '.attempts // 0')
  challenge_type=$(echo "$RESULT" | jq -r '.challengeType // ""')
  echo -e "  ${MUTED}solver=${solver_name:-auto} solved=${solved} attempts=${attempts} challengeType=${challenge_type}${NC}"

  sleep 2

  local after_text
  after_text=$(extract_text_for_tab "$tab_id")
  if [ "$HTTP_STATUS" != "200" ]; then
    echo -e "  ${RED}✗${NC} failed to extract post-solve text (${HTTP_STATUS})"
    ((ASSERTIONS_FAILED++)) || true
    return
  fi

  local after_signals
  after_signals=$(capture_signals_json "$after_text")

  log_signal_delta "Detections (before -> after):" "$before_signals" "$after_signals"

  # The test is observational: success means pipeline completed and signals were captured.
  ((ASSERTIONS_PASSED++)) || true
}

start_test "TestAutoSolver_RealWorldSimulation"

split_csv_targets "$TARGETS_CSV"
for target in "${TARGETS[@]}"; do
  target="${target## }"
  target="${target%% }"
  if [ -z "$target" ]; then
    continue
  fi
  run_site "$target"
done

end_test

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  print_summary
fi
