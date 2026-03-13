#!/bin/bash
# 13-pdf.sh — CLI PDF export command

source "$(dirname "$0")/common.sh"

# SKIP: pdf needs -o flag but cobra eats unknown flags
# Binary stdout doesn't work well with pt() text capture
# Needs cobra flag registration for -o, --tab, etc.
