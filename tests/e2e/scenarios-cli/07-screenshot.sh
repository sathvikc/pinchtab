#!/bin/bash
# 07-screenshot.sh — CLI screenshot command

source "$(dirname "$0")/common.sh"

# SKIP: screenshot needs -o flag but cobra eats unknown flags
# Binary stdout also doesn't work well with pt() text capture
# Needs cobra flag registration for -o, -q, --tab
