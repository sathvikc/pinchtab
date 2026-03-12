#!/bin/bash
# 16-meta.sh — CLI meta commands (version, help)

source "$(dirname "$0")/common.sh"

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab --version"

pt_ok --version
assert_output_contains "pinchtab" "outputs version string"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab help"

pt_ok help
assert_output_contains "pinchtab" "outputs help text"
assert_output_contains "nav" "mentions nav command"

end_test

# ─────────────────────────────────────────────────────────────────
start_test "pinchtab --help"

pt_ok --help
assert_output_contains "pinchtab" "outputs help text"

end_test
