#!/bin/bash
# orchestrator-extended.sh — API full orchestration scenarios.

GROUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${GROUP_DIR}/../../helpers/api.sh"

BRIDGE_URL="${E2E_BRIDGE_URL:-}"
BRIDGE_TOKEN="${E2E_BRIDGE_TOKEN:-}"

if [ -z "$BRIDGE_URL" ]; then
  echo "  E2E_BRIDGE_URL not set, skipping orchestrator full scenarios"
  return 0 2>/dev/null || exit 0
fi

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: attach remote bridge and proxy tab traffic"

pt_post /instances/attach-bridge "{\"name\":\"e2e-remote-bridge\",\"baseUrl\":\"${BRIDGE_URL}\",\"token\":\"${BRIDGE_TOKEN}\"}"
assert_http_status "201" "attach bridge"
assert_json_eq "$RESULT" '.attachType' 'bridge' "instance attachType is bridge"
assert_json_eq "$RESULT" '.attached' 'true' "instance is marked attached"
assert_json_eq "$RESULT" '.url' "${BRIDGE_URL}" "instance stores remote bridge URL"

ATTACHED_INST_ID=$(echo "$RESULT" | jq -r '.id // empty')
if [ -n "$ATTACHED_INST_ID" ]; then
  echo -e "  ${GREEN}✓${NC} attached bridge instance id: ${ATTACHED_INST_ID}"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} attach response missing instance id"
  ((ASSERTIONS_FAILED++)) || true
fi

pt_post "/instances/${ATTACHED_INST_ID}/tabs/open" "{\"url\":\"${FIXTURES_URL}/index.html\"}"
assert_ok "open tab on attached bridge"
assert_tab_id "attached bridge returned tabId"
ATTACHED_TAB_ID="${TAB_ID}"

pt_get "/tabs/${ATTACHED_TAB_ID}/text?format=text"
assert_ok "proxy text via attached bridge tab route"
assert_contains "$RESULT" "Welcome to the E2E test fixtures." "tab text came back through orchestrator proxy"

pt_get "/instances/${ATTACHED_INST_ID}/tabs"
assert_ok "list tabs for attached bridge instance"
assert_json_length_gte "$RESULT" '.' '1' "attached bridge has at least one tab"

pt_get /instances/tabs
assert_ok "aggregate tabs includes attached bridge"
if echo "$RESULT" | jq -e --arg inst "$ATTACHED_INST_ID" '.[] | select(.instanceId == $inst)' >/dev/null 2>&1; then
  echo -e "  ${GREEN}✓${NC} aggregate tab list includes attached bridge instance"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} aggregate tab list missing attached bridge instance"
  ((ASSERTIONS_FAILED++)) || true
fi

pt_post "/instances/${ATTACHED_INST_ID}/stop" '{}'
assert_ok "stop attached bridge instance"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: health shows dashboard mode"

pt_get /health
assert_ok "health"
assert_json_eq "$RESULT" '.mode' 'dashboard'

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: list instances"

pt_get /instances
assert_ok "list instances"
assert_json_length_gte "$RESULT" '.' '1' "at least 1 instance"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: launch new instance"

pt_post /instances/start '{"mode":"headless"}'
assert_ok "launch instance"
assert_json_eq "$RESULT" '.mode' 'headless' "launch response includes mode"
assert_json_eq "$RESULT" '.headless' 'true' "launch response keeps headless boolean"

INST_ID=$(echo "$RESULT" | jq -r '.id')
assert_json_exists "$RESULT" '.id' "has instance id"
assert_json_exists "$RESULT" '.port' "has port"

wait_for_orchestrator_instance_status "${E2E_SERVER}" "${INST_ID}" "running" 30

pt_get /instances
assert_ok "list after launch"
assert_instance_list_contains "$INST_ID" "instance $INST_ID in list" "instance $INST_ID not in list"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: get instance by id"

pt_get "/instances/${INST_ID}"
assert_ok "get instance"
assert_json_eq "$RESULT" '.id' "$INST_ID"
assert_json_eq "$RESULT" '.mode' 'headless' "instance response includes mode"
assert_json_eq "$RESULT" '.headless' 'true' "instance response keeps headless boolean"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: instance logs"

pt_get "/instances/${INST_ID}/logs"
assert_ok "get logs"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: aggregate tabs"

pt_post /navigate "{\"url\":\"${FIXTURES_URL}/index.html\"}"
assert_ok "navigate"

pt_get /instances/tabs
assert_ok "aggregate tabs"
assert_json_length_gte "$RESULT" '.' '1' "at least 1 tab"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: aggregate tabs (multi-instance)"

wait_for_orchestrator_instance_status "${E2E_SERVER}" "${INST_ID}" "running" 10

pt_post /navigate "{\"url\":\"${FIXTURES_URL}/index.html\"}"
assert_ok "navigate on default instance"
pt_post "/instances/${INST_ID}/tabs/open" "{\"url\":\"${FIXTURES_URL}/form.html\"}"
assert_ok "open tab on launched instance"

pt_get /instances/tabs
assert_ok "aggregate tabs"
assert_json_length_gte "$RESULT" '.' '2' "at least 2 tabs across instances"

end_test

# Note: the next three tests intentionally depend on INST_ID from "launch new instance".
# They validate the same launched-instance lifecycle in sequence:
# aggregate across instances, inspect its tabs directly, then stop it.
# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: instance tabs"

wait_for_orchestrator_instance_status "${E2E_SERVER}" "${INST_ID}" "running" 10

pt_get "/instances/${INST_ID}/tabs"
assert_ok "instance tabs"

end_test

# ─────────────────────────────────────────────────────────────────
# Run the ID-format check before the stop test so it can reuse the
# still-running INST_ID from "launch new instance" instead of paying
# another full launch+stop cycle (~6s) just to inspect the prefix.
start_test "orchestrator: ID format (inst_ prefix)"

assert_instance_id_prefix "$INST_ID"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: stop instance"

pt_post "/instances/${INST_ID}/stop" '{}'
assert_ok "stop instance"

wait_for_instances_gone "${E2E_SERVER}" 10 "${INST_ID}" || true
pt_get /instances
assert_instance_list_absent "$INST_ID" "instance removed after stop" "instance still in list after stop"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: ports, isolation, and cleanup"

ACTIVE_INST_IDS=()

# Each /instances/start is synchronous (blocks until Chrome is up,
# ~2-5s). Issue independent launches in parallel via background curl
# so the test waits for max(launch_time) instead of sum(launch_time).
parallel_post() {
  # parallel_post out_file body — fires one POST /instances/start
  e2e_curl -s -w "\n%{http_code}" \
    -X POST "${E2E_SERVER}/instances/start" \
    -H "Content-Type: application/json" \
    -d "$2" > "$1"
}
read_resp() {
  # read_resp out_file body_var status_var
  local content
  content=$(cat "$1")
  printf -v "$3" '%s' "${content##*$'\n'}"
  printf -v "$2" '%s' "${content%$'\n'*}"
}
record_assertion() {
  if [[ "$1" =~ ^2 ]]; then
    echo -e "  ${GREEN}✓${NC} $2 → $1"
    ((ASSERTIONS_PASSED++)) || true
  else
    echo -e "  ${RED}✗${NC} $2 → $1"
    ((ASSERTIONS_FAILED++)) || true
  fi
}

OUT1=$(mktemp); OUT2=$(mktemp)
parallel_post "$OUT1" '{"mode":"headless"}' &
parallel_post "$OUT2" '{"mode":"headless"}' &
wait
read_resp "$OUT1" RESP1 CODE1
read_resp "$OUT2" RESP2 CODE2
rm -f "$OUT1" "$OUT2"
record_assertion "$CODE1" "launch 1"
record_assertion "$CODE2" "launch 2"
INST1=$(echo "$RESP1" | jq -r '.id')
PORT1=$(echo "$RESP1" | jq -r '.port')
INST2=$(echo "$RESP2" | jq -r '.id')
PORT2=$(echo "$RESP2" | jq -r '.port')
ACTIVE_INST_IDS+=("$INST1" "$INST2")

if [ "$PORT1" != "$PORT2" ]; then
  echo -e "  ${GREEN}✓${NC} unique ports: $PORT1 vs $PORT2"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} duplicate ports: $PORT1"
  ((ASSERTIONS_FAILED++)) || true
fi

pt_post "/instances/${INST1}/stop" '{}'
assert_ok "stop first instance"
wait_for_instances_gone "${E2E_SERVER}" 10 "${INST1}" || true
ACTIVE_INST_IDS=("${INST2}")

pt_post /instances/start '{"mode":"headless"}'
assert_ok "relaunch"
INST3=$(echo "$RESULT" | jq -r '.id')
PORT3=$(echo "$RESULT" | jq -r '.port')
ACTIVE_INST_IDS+=("$INST3")

if [ "$PORT1" = "$PORT3" ]; then
  echo -e "  ${GREEN}✓${NC} port reused: $PORT1"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${YELLOW}⚠${NC} port not reused ($PORT1 vs $PORT3) — may depend on timing"
  ((ASSERTIONS_PASSED++)) || true
fi

if wait_for_instances_running "${E2E_SERVER}" 30 "${INST2}" "${INST3}"; then
  echo -e "  ${GREEN}✓${NC} reused instances are running"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} reused instances did not both reach running"
  ((ASSERTIONS_FAILED++)) || true
fi

# Open tabs on independent instances in parallel.
TAB_OUT1=$(mktemp); TAB_OUT2=$(mktemp)
e2e_curl -s -w "\n%{http_code}" \
  -X POST "${E2E_SERVER}/instances/${INST2}/tabs/open" \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"${FIXTURES_URL}/index.html\"}" > "$TAB_OUT1" &
e2e_curl -s -w "\n%{http_code}" \
  -X POST "${E2E_SERVER}/instances/${INST3}/tabs/open" \
  -H "Content-Type: application/json" \
  -d "{\"url\":\"${FIXTURES_URL}/form.html\"}" > "$TAB_OUT2" &
wait
read_resp "$TAB_OUT1" TAB_RESP1 TAB_CODE1
read_resp "$TAB_OUT2" TAB_RESP2 TAB_CODE2
rm -f "$TAB_OUT1" "$TAB_OUT2"
record_assertion "$TAB_CODE1" "open tab on second instance"
record_assertion "$TAB_CODE2" "open tab on reused instance"
TAB1=$(echo "$TAB_RESP1" | jq -r '.tabId // .id // empty')
TAB2=$(echo "$TAB_RESP2" | jq -r '.tabId // .id // empty')

if [ -n "$TAB1" ] && [ -n "$TAB2" ] && [ "$TAB1" != "$TAB2" ]; then
  echo -e "  ${GREEN}✓${NC} instances have separate tabs"
  ((ASSERTIONS_PASSED++)) || true
else
  echo -e "  ${RED}✗${NC} instances did not produce distinct tab IDs: ${TAB1} vs ${TAB2}"
  ((ASSERTIONS_FAILED++)) || true
fi

pt_post /instances/start '{"mode":"headless"}'
assert_ok "launch cleanup-3"
INST4=$(echo "$RESULT" | jq -r '.id')
ACTIVE_INST_IDS+=("$INST4")

# Stop all instances in parallel; the wait_for_instances_gone below
# already polls for completion, so individual stop responses just need
# to be 2xx.
declare -a STOP_OUTS=()
for id in "${ACTIVE_INST_IDS[@]}"; do
  out=$(mktemp)
  STOP_OUTS+=("$out")
  e2e_curl -s -w "\n%{http_code}" \
    -X POST "${E2E_SERVER}/instances/${id}/stop" \
    -H "Content-Type: application/json" \
    -d '{}' > "$out" &
done
wait
for i in "${!ACTIVE_INST_IDS[@]}"; do
  read_resp "${STOP_OUTS[$i]}" _ STOP_CODE
  record_assertion "$STOP_CODE" "stop ${ACTIVE_INST_IDS[$i]}"
  rm -f "${STOP_OUTS[$i]}"
done

wait_for_instances_gone "${E2E_SERVER}" 10 "${ACTIVE_INST_IDS[@]}" || true

pt_get /instances
for id in "${ACTIVE_INST_IDS[@]}"; do
  FOUND=$(echo "$RESULT" | jq -r ".[] | select(.id == \"$id\") | .id")
  if [ -z "$FOUND" ] || [ "$FOUND" = "null" ]; then
    echo -e "  ${GREEN}✓${NC} $id removed"
    ((ASSERTIONS_PASSED++)) || true
  else
    echo -e "  ${RED}✗${NC} $id still present"
    ((ASSERTIONS_FAILED++)) || true
  fi
done

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: proxy with query params"

pt_post /navigate '{"url":"'"${FIXTURES_URL}"'/index.html?foo=bar&baz=qux"}'
assert_ok "navigate with query params"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: stop non-existent instance"

pt_post "/instances/nonexistent_xyz/stop" '{}'
assert_not_ok "rejects bad instance id"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "orchestrator: proxy routing"

pt_post /navigate "{\"url\":\"${FIXTURES_URL}/buttons.html\"}"
assert_ok "navigate via proxy"

pt_get /snapshot
assert_ok "snapshot via proxy"
assert_json_exists "$RESULT" '.nodes' "has nodes"

end_test
