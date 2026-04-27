// Package main implements the benchmark runner.
//
// This file defines the command-line interface. Two flags, --dry-run and
// --index-file, allow exercising the plan/prompt assembly without needing
// network access or the default index.md location.
package bench

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Lane is the benchmark lane (pinchtab or agent-browser).
type Lane string

const (
	LanePinchtab     Lane = "pinchtab"
	LaneAgentBrowser Lane = "agent-browser"
)

// Provider identifies which API the runner talks to.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenAI    Provider = "openai"
	ProviderFake      Provider = "fake"
	ProviderUnset     Provider = ""
)

// Args is the resolved plan for a single invocation. It is intentionally a
// plain-data struct so tests can construct one without touching os.Args.
type Args struct {
	Lane            Lane
	Provider        Provider
	Model           string
	Groups          []int
	Profile         string
	MaxTokens       int
	Temperature     float64
	MaxTurns        int
	MaxIdleTurns    int
	TimeoutSeconds  int
	TurnDelayMs     int
	ReportFile      string
	SkipInit        bool
	NoPromptCaching bool
	Finalize        bool
	DryRun          bool
	Verbose         bool
	IndexFile       string
	MaxInputTokens  int
	MaxOutputTokens int
	TerseSummary    bool
}

// Defaults matches the literal defaults in the TypeScript runner. Keep this in
// sync with parseArgs() in run-api-benchmark.ts.
func defaultArgs() Args {
	return Args{
		MaxTokens:      4096,
		Temperature:    0,
		MaxTurns:       300,
		MaxIdleTurns:   25,
		TimeoutSeconds: 120,
		TurnDelayMs:    1500,
	}
}

const usageText = `Usage:
  runner --lane pinchtab [options]     Run benchmark loop
  runner --lane agent-browser [options]
  runner step-end [options] <group> <step> <status> <answer> <verify-status> <notes>
  runner record-step [options] <group> <step> <status> [answer] [notes]
  runner verify-step [options] <group> <step> <status> [notes]

Subcommand options:
  --type baseline|pinchtab|agent-browser
  --report-file PATH

Benchmark loop options:
  --provider anthropic|openai|fake
  --model MODEL
  --groups 0,1,2,3
  --profile common10
  --max-tokens N
  --temperature N
  --max-turns N
  --max-idle-turns N
  --timeout-seconds N
  --turn-delay-ms N
  --report-file PATH
  --skip-init
  --no-prompt-caching
  --finalize
  --dry-run                 Print the resolved plan and exit 0 without network access
  --index-file PATH         Override path to tests/benchmark/index.md
  --max-input-tokens N      Stop when cumulative input tokens exceed N (exit code 4)
  --max-output-tokens N     Stop when cumulative output tokens exceed N (exit code 4)
  --terse-summary           Tell the agent to end with "done" (no prose summary); use for benchmark runs
  --verbose, -v             Show detailed progress with spinners
`

// ParseArgs walks argv manually (like the TS runner) so the flag surface and
// error messages stay byte-identical. The stdlib `flag` package would reorder
// output and reject `--groups 0,1,2` style values in some edge cases.
func ParseArgs(argv []string) (Args, error) {
	a := defaultArgs()

	next := func(i *int, name string) (string, error) {
		*i++
		if *i >= len(argv) {
			return "", fmt.Errorf("%s requires a value", name)
		}
		return argv[*i], nil
	}

	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		switch arg {
		case "--lane":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.Lane = Lane(v)
		case "--provider":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.Provider = Provider(v)
		case "--model":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.Model = v
		case "--groups":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			groups, perr := parseGroups(v)
			if perr != nil {
				return a, perr
			}
			a.Groups = groups
		case "--profile":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.Profile = v
		case "--max-tokens":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--max-tokens: %w", perr)
			}
			a.MaxTokens = n
		case "--temperature":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			f, perr := strconv.ParseFloat(v, 64)
			if perr != nil {
				return a, fmt.Errorf("--temperature: %w", perr)
			}
			a.Temperature = f
		case "--max-turns":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--max-turns: %w", perr)
			}
			a.MaxTurns = n
		case "--max-idle-turns":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--max-idle-turns: %w", perr)
			}
			a.MaxIdleTurns = n
		case "--timeout-seconds":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--timeout-seconds: %w", perr)
			}
			a.TimeoutSeconds = n
		case "--turn-delay-ms":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--turn-delay-ms: %w", perr)
			}
			a.TurnDelayMs = n
		case "--report-file":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.ReportFile = v
		case "--skip-init":
			a.SkipInit = true
		case "--no-prompt-caching":
			a.NoPromptCaching = true
		case "--finalize":
			a.Finalize = true
		case "--terse-summary":
			a.TerseSummary = true
		case "--dry-run":
			a.DryRun = true
		case "--verbose", "-v":
			a.Verbose = true
		case "--index-file":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			a.IndexFile = v
		case "--max-input-tokens":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--max-input-tokens: %w", perr)
			}
			a.MaxInputTokens = n
		case "--max-output-tokens":
			v, err := next(&i, arg)
			if err != nil {
				return a, err
			}
			n, perr := strconv.Atoi(v)
			if perr != nil {
				return a, fmt.Errorf("--max-output-tokens: %w", perr)
			}
			a.MaxOutputTokens = n
		case "-h", "--help", "help":
			return a, errHelp
		default:
			return a, fmt.Errorf("unknown argument: %s", arg)
		}
	}

	if a.Lane != LanePinchtab && a.Lane != LaneAgentBrowser {
		return a, errors.New("--lane must be 'pinchtab' or 'agent-browser'")
	}

	return a, nil
}

// errHelp is a sentinel the caller uses to distinguish "user asked for help"
// (exit 0) from "bad flags" (exit 1). It matches the TS runner's usage(0) vs
// usage(1) behaviour.
var errHelp = errors.New("help requested")

// parseGroups splits "0,1,2,3" into sorted unique ints, matching the TS
// implementation's `.map(Number).filter(Number.isInteger)` semantics.
func parseGroups(raw string) ([]int, error) {
	seen := make(map[int]struct{})
	var out []int
	for _, piece := range strings.Split(raw, ",") {
		piece = strings.TrimSpace(piece)
		if piece == "" {
			continue
		}
		n, err := strconv.Atoi(piece)
		if err != nil {
			// TS filters silently; we do too, to preserve parity.
			continue
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}

// WriteUsage prints the help block to the given writer. Kept as a helper so
// tests can capture output without redirecting os.Stderr.
func WriteUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}
