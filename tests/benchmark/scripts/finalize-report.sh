#!/usr/bin/env bash
#
# Renders a short markdown summary for the latest benchmark JSON report.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCH_DIR="${SCRIPT_DIR}/.."
RESULTS_DIR="${BENCH_DIR}/results"

REPORT_FILE="${1:-}"
if [[ -z "${REPORT_FILE}" ]]; then
  shopt -s nullglob
  candidates=(
    "${RESULTS_DIR}"/agent_browser_benchmark_*.json
    "${RESULTS_DIR}"/agent_benchmark_*.json
    "${RESULTS_DIR}"/baseline_*.json
  )
  shopt -u nullglob

  if [[ ${#candidates[@]} -gt 0 ]]; then
    REPORT_FILE=$(ls -t "${candidates[@]}" | head -1)
  fi
fi

if [[ -z "${REPORT_FILE}" || ! -f "${REPORT_FILE}" ]]; then
  echo "ERROR: no benchmark report found"
  exit 1
fi

SUMMARY_FILE="${REPORT_FILE%.json}_summary.md"

jq -r '
  def pct($a; $b):
    if $b == 0 then "0.0%" else (((1000 * $a) / $b | round) / 10 | tostring) + "%" end;
  . as $root |
  ($root.totals.steps_passed + $root.totals.steps_failed + $root.totals.steps_skipped) as $step_count |
  "# Benchmark Summary",
  "",
  "| Metric | Value |",
  "|--------|-------|",
  "| Type | \($root.benchmark.type) |",
  "| Model | \($root.benchmark.model) |",
  "| Steps Passed | \($root.totals.steps_passed) |",
  "| Steps Failed | \($root.totals.steps_failed) |",
  "| Steps Skipped | \($root.totals.steps_skipped) |",
  "| Pass Rate | \(pct($root.totals.steps_passed; $step_count)) |",
  "| Input Tokens | \($root.totals.input_tokens) |",
  "| Output Tokens | \($root.totals.output_tokens) |",
  "| Total Tokens | \($root.totals.total_tokens) |",
  "| Tool Calls | \($root.totals.tool_calls // 0) |",
  "| Estimated Cost (USD) | \($root.totals.estimated_cost_usd) |",
  "",
  "## Failed Steps",
  "",
  (
    [ $root.steps[] | select(.status == "fail") | "- \(.id): \(.notes)" ] |
    if length == 0 then ["- none"] else . end
  )[]
' "${REPORT_FILE}" > "${SUMMARY_FILE}"

echo "Wrote ${SUMMARY_FILE}"
