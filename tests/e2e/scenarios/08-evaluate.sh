#!/bin/bash
# 08-evaluate.sh — JavaScript evaluation

source "$(dirname "$0")/common.sh"

# Navigate to evaluate test page
pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/evaluate.html\"}"
sleep 1

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate (simple expression)"

pt_post /evaluate -d '{"expression":"1 + 1"}'
assert_ok "evaluate simple"

# Verify result
RESULT=$(echo "$LAST_BODY" | jq -r '.result')
if [ "$RESULT" != "2" ]; then
  fail "expected result=2, got $RESULT"
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate (DOM query)"

pt_post /evaluate -d '{"expression":"document.title"}'
assert_ok "evaluate DOM"

RESULT=$(echo "$LAST_BODY" | jq -r '.result')
if [ "$RESULT" != "Evaluate Test Page" ]; then
  fail "expected title, got $RESULT"
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate (call function)"

pt_post /evaluate -d '{"expression":"window.calculate.add(5, 3)"}'
assert_ok "evaluate function"

RESULT=$(echo "$LAST_BODY" | jq -r '.result')
if [ "$RESULT" != "8" ]; then
  fail "expected result=8, got $RESULT"
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate (get object)"

pt_post /evaluate -d '{"expression":"JSON.stringify(window.testData)"}'
assert_ok "evaluate object"

# Verify we got the test data
if ! echo "$LAST_BODY" | jq -r '.result' | jq -e '.name == "PinchTab"' >/dev/null 2>&1; then
  fail "expected testData object"
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate (modify DOM)"

pt_post /evaluate -d '{"expression":"document.getElementById(\"counter\").textContent = \"42\"; 42"}'
assert_ok "evaluate modify DOM"

# Verify the change stuck
pt_post /evaluate -d '{"expression":"document.getElementById(\"counter\").textContent"}'
RESULT=$(echo "$LAST_BODY" | jq -r '.result')
if [ "$RESULT" != "42" ]; then
  fail "expected counter=42, got $RESULT"
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab evaluate --tab <id>"

# Open a new tab with evaluate page to get a known tab ID
pt_post /navigate -d "{\"url\":\"${FIXTURES_URL}/evaluate.html\",\"newTab\":true}"
sleep 1

pt_get /tabs
TAB_ID=$(get_last_tab)

pt_post "/tabs/${TAB_ID}/evaluate" -d '{"expression":"1 + 2 + 3"}'
assert_ok "tab evaluate"

RESULT=$(echo "$LAST_BODY" | jq -r '.result')
if [ "$RESULT" != "6" ]; then
  fail "expected result=6, got $RESULT"
fi

end_test
