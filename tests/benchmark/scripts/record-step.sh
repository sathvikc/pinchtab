#!/bin/bash
#
# Record a benchmark step result
#
# Usage:
#   ./record-step.sh [--type baseline|agent|agent-browser] <group> <step> <pass|fail|skip> [options] "notes"
#
# Options:
#   --type baseline|agent|agent-browser   Report type (default: auto-detect most recent)
#   --tokens <in> <out>     Token usage (agent runs only, default: 0 0)
#   --bytes <n>             HTTP response size in bytes (baseline runs)
#   --tool-calls <n>        Override auto-counted tool calls
#
# Examples:
#   ./record-step.sh 1 1 pass "Navigation completed"
#   ./record-step.sh --type agent 2 3 fail --tokens 200 80 "Element not found"
#   ./record-step.sh --type baseline 1 2 pass --bytes 4520 "Snapshot returned"

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESULTS_DIR="${SCRIPT_DIR}/../results"
mkdir -p "${RESULTS_DIR}"

# Parse flags
REPORT_TYPE=""
INPUT_TOKENS=0
OUTPUT_TOKENS=0
RESPONSE_BYTES=0
TOOL_CALLS=""
AGENT_BROWSER_LOG="${RESULTS_DIR}/agent_browser_commands.ndjson"

while [[ $# -gt 0 && "$1" == --* ]]; do
    case "$1" in
        --type)
            REPORT_TYPE="$2"
            shift 2
            ;;
        --tokens)
            INPUT_TOKENS="$2"
            OUTPUT_TOKENS="$3"
            shift 3
            ;;
        --bytes)
            RESPONSE_BYTES="$2"
            shift 2
            ;;
        --tool-calls)
            TOOL_CALLS="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

if [[ $# -lt 3 ]]; then
    echo "Usage: $0 [--type baseline|agent|agent-browser] <group> <step> <pass|fail|skip> [--tokens <in> <out>] [--bytes <n>] [--tool-calls <n>] [notes]"
    exit 1
fi

GROUP="$1"
STEP="$2"
STATUS="$3"
NOTES="${4:-}"
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
TOTAL_TOKENS=$((INPUT_TOKENS + OUTPUT_TOKENS))

# Find report file
if [[ -n "${REPORT_TYPE}" ]]; then
    case "${REPORT_TYPE}" in
        baseline)
            REPORT_FILE=$(ls -t "${RESULTS_DIR}"/baseline_*.json 2>/dev/null | head -1)
            ;;
        agent)
            REPORT_FILE=$(ls -t "${RESULTS_DIR}"/agent_benchmark_*.json 2>/dev/null | head -1)
            ;;
        agent-browser|agent_browser)
            REPORT_FILE=$(ls -t "${RESULTS_DIR}"/agent_browser_benchmark_*.json 2>/dev/null | head -1)
            ;;
        *)
            echo "ERROR: --type must be 'baseline', 'agent', or 'agent-browser'"
            exit 1
            ;;
    esac
else
    # Auto-detect: find most recent report of any type
    REPORT_FILE=$(ls -t "${RESULTS_DIR}"/baseline_*.json "${RESULTS_DIR}"/agent_benchmark_*.json "${RESULTS_DIR}"/agent_browser_benchmark_*.json 2>/dev/null | head -1)
fi

if [[ -z "${REPORT_FILE}" ]]; then
    echo "ERROR: No benchmark report found. Run ./run-optimization.sh first."
    exit 1
fi

STEP_TOOL_CALLS=0
if [[ -n "${TOOL_CALLS}" ]]; then
    STEP_TOOL_CALLS="${TOOL_CALLS}"
elif [[ "$(jq -r '.benchmark.type' "${REPORT_FILE}")" == "agent-browser" ]]; then
    CURRENT_TOOL_CALLS=0
    PREV_TOOL_CALLS=$(jq -r '.totals.tool_calls // 0' "${REPORT_FILE}")
    if [[ -f "${AGENT_BROWSER_LOG}" ]]; then
        CURRENT_TOOL_CALLS=$(jq -Rsc 'split("\n") | map(select(length > 0) | try fromjson catch empty) | length' "${AGENT_BROWSER_LOG}")
    fi
    if [[ "${CURRENT_TOOL_CALLS}" -ge "${PREV_TOOL_CALLS}" ]]; then
        STEP_TOOL_CALLS=$((CURRENT_TOOL_CALLS - PREV_TOOL_CALLS))
    fi
fi

# Calculate cost (only meaningful for agent runs with token data)
COST=0
if [[ ${TOTAL_TOKENS} -gt 0 ]]; then
    MODEL=$(jq -r '.benchmark.model' "${REPORT_FILE}")

    # Cost per 1M tokens
    case "${MODEL}" in
        *haiku*) INPUT_RATE=0.25; OUTPUT_RATE=1.25 ;;
        *sonnet*) INPUT_RATE=3.0; OUTPUT_RATE=15.0 ;;
        *opus*) INPUT_RATE=15.0; OUTPUT_RATE=75.0 ;;
        *gpt-4o-mini*) INPUT_RATE=0.15; OUTPUT_RATE=0.60 ;;
        *gpt-4o*) INPUT_RATE=2.50; OUTPUT_RATE=10.0 ;;
        *gpt-4*) INPUT_RATE=10.0; OUTPUT_RATE=30.0 ;;
        *gemini*flash*) INPUT_RATE=0.075; OUTPUT_RATE=0.30 ;;
        *gemini*pro*) INPUT_RATE=1.25; OUTPUT_RATE=5.0 ;;
        *) INPUT_RATE=1.0; OUTPUT_RATE=3.0 ;;
    esac

    COST=$(echo "scale=6; (${INPUT_TOKENS} / 1000000 * ${INPUT_RATE}) + (${OUTPUT_TOKENS} / 1000000 * ${OUTPUT_RATE})" | bc)
fi

# Create step entry
STEP_JSON=$(jq -n \
    --argjson group "${GROUP}" \
    --argjson step "${STEP}" \
    --arg id "${GROUP}.${STEP}" \
    --arg status "${STATUS}" \
    --argjson in_tokens "${INPUT_TOKENS}" \
    --argjson out_tokens "${OUTPUT_TOKENS}" \
    --argjson total_tokens "${TOTAL_TOKENS}" \
    --argjson tool_calls "${STEP_TOOL_CALLS}" \
    --argjson cost "${COST}" \
    --argjson bytes "${RESPONSE_BYTES}" \
    --arg notes "${NOTES}" \
    --arg ts "${TIMESTAMP}" \
    '{group: $group, step: $step, id: $id, status: $status,
      input_tokens: $in_tokens, output_tokens: $out_tokens,
      total_tokens: $total_tokens, tool_calls: $tool_calls,
      cost_usd: $cost,
      response_bytes: $bytes, notes: $notes, timestamp: $ts}')

# Append to report and update totals
TMP_FILE=$(mktemp)
jq --argjson step "${STEP_JSON}" \
   --argjson in "${INPUT_TOKENS}" \
   --argjson out "${OUTPUT_TOKENS}" \
   --argjson tool_calls "${STEP_TOOL_CALLS}" \
   --argjson cost "${COST}" \
   --arg status "${STATUS}" \
   '(.totals.tool_calls //= 0) |
    .steps += [$step] |
    .totals.input_tokens += $in |
    .totals.output_tokens += $out |
    .totals.total_tokens += ($in + $out) |
    .totals.tool_calls += $tool_calls |
    .totals.estimated_cost_usd += $cost |
    if $status == "pass" then .totals.steps_passed += 1
    elif $status == "fail" then .totals.steps_failed += 1
    else .totals.steps_skipped += 1 end' \
   "${REPORT_FILE}" > "${TMP_FILE}"

mv "${TMP_FILE}" "${REPORT_FILE}"

# Log failures
if [[ "${STATUS}" == "fail" ]]; then
    echo "[${TIMESTAMP}] Step ${GROUP}.${STEP} FAILED: ${NOTES}" >> "${RESULTS_DIR}/errors.log"
fi

echo "Recorded: Step ${GROUP}.${STEP} = ${STATUS} (tokens: ${TOTAL_TOKENS}, bytes: ${RESPONSE_BYTES}, \$${COST})"
