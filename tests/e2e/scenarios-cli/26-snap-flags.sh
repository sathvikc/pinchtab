#!/bin/bash
# 26-snap-flags.sh — CLI snapshot flags (previously blocked by cobra)

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --text"

pt_ok nav "${FIXTURES_URL}/index.html"
pt_ok snap --text

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --interactive"

pt_ok snap --interactive

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --compact"

pt_ok snap --compact

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --depth 2"

pt_ok snap --depth 2

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --max-tokens 100"

pt_ok snap --max-tokens 100

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap --diff"

pt_ok snap
assert_exit_ok "first snapshot"
pt_ok snap --diff
assert_exit_ok "snap diff mode"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab snap -s 'body'"

pt_ok snap -s "body"
assert_exit_ok "snap with selector"

end_test
