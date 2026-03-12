#!/bin/bash
# 17-tabs-ops.sh — Tab-specific operations (snapshot, screenshot, eval, cookies)

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tabs snapshot <tabId>"

pt_ok nav "${FIXTURES_URL}/form.html"
TAB_ID=$(echo "$PT_OUT" | jq -r '.tabId')

pt_ok tabs snapshot "$TAB_ID"
assert_output_json
assert_output_contains "nodes" "returns snapshot nodes"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tabs screenshot <tabId>"

TMPFILE="/tmp/test-tab-ss-$$.jpg"
pt_ok tabs screenshot "$TAB_ID" -o "$TMPFILE"

if [ -f "$TMPFILE" ] && [ -s "$TMPFILE" ]; then
  echo -e "  ${GREEN}✓${NC} tab screenshot saved"
  ((ASSERTIONS_PASSED++)) || true
  rm -f "$TMPFILE"
else
  echo -e "  ${RED}✗${NC} tab screenshot not created"
  ((ASSERTIONS_FAILED++)) || true
fi

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tabs eval <tabId> <expression>"

pt_ok tabs eval "$TAB_ID" "document.title"
assert_output_contains "result" "returns eval result"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tabs cookies <tabId>"

pt_ok tabs cookies "$TAB_ID"
assert_output_json
# Cookies array may be empty but should be valid JSON

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab tabs text <tabId>"

pt_ok tabs text "$TAB_ID"
assert_output_json
assert_output_contains "text" "returns text content"

end_test
