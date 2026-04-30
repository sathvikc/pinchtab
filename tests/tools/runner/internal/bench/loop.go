package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type LoopConfig struct {
	Lane            Lane
	Provider        string
	Model           string
	Groups          []int
	ReportFile      string
	MaxTurns        int
	MaxIdleTurns    int
	TimeoutSeconds  int
	TurnDelayMs     int
	Finalize        bool
	SkipInit        bool
	ToolsDir        string
	BenchmarkDir    string
	CommandLogFile  string
	MaxInputTokens  int
	MaxOutputTokens int
	TerseSummary    bool
	Verbose         bool
	Stdout          io.Writer
	Stderr          io.Writer
}

type LoopResult struct {
	ExitCode  int
	FinalText string
	Usage     UsageCounters
}

func RunLoop(cfg LoopConfig, runner Runner, shell *PersistentShell) LoopResult {
	out := NewOutputWriter(cfg.Stdout, cfg.Verbose)

	promptCfg := PromptConfig{
		RepoRoot:     filepath.Dir(filepath.Dir(cfg.ToolsDir)),
		ToolsDir:     cfg.ToolsDir,
		BenchmarkDir: cfg.BenchmarkDir,
		TerseSummary: cfg.TerseSummary,
	}

	laneCfg := LanePromptConfig(cfg.Lane, promptCfg)

	// Download agent-browser skill if not cached
	if cfg.Lane == LaneAgentBrowser && laneCfg.SkillFile == "" {
		skillFile, err := DownloadAgentBrowserSkill(promptCfg.RepoRoot, cfg.ToolsDir)
		if err != nil {
			out.FinalText(fmt.Sprintf("Failed to download agent-browser skill: %v", err))
			return LoopResult{ExitCode: 1, FinalText: err.Error()}
		}
		laneCfg.SkillFile = skillFile
	}

	userPrompt := LaneUserPrompt(cfg.Lane, promptCfg, cfg.ReportFile, cfg.Groups)
	conversation := runner.InitialConversation(userPrompt)
	systemPromptText := SystemPrompt(laneCfg)

	var finalText string
	exitCode := 0
	idleTurns := 0
	lastAnswered := ReadProgress(cfg.ReportFile).Answered

	clearCommandLog(cfg.CommandLogFile)

	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		if turn > 1 && cfg.TurnDelayMs > 0 {
			time.Sleep(time.Duration(cfg.TurnDelayMs) * time.Millisecond)
		}

		usage := runner.Usage()
		if cfg.MaxInputTokens > 0 && usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens >= cfg.MaxInputTokens {
			finalText = fmt.Sprintf("Budget exceeded: input tokens (%d) >= limit (%d)", usage.InputTokens+usage.CacheCreationInputTokens+usage.CacheReadInputTokens, cfg.MaxInputTokens)
			exitCode = 4
			break
		}
		if cfg.MaxOutputTokens > 0 && usage.OutputTokens >= cfg.MaxOutputTokens {
			finalText = fmt.Sprintf("Budget exceeded: output tokens (%d) >= limit (%d)", usage.OutputTokens, cfg.MaxOutputTokens)
			exitCode = 4
			break
		}

		out.Turn(turn, cfg.MaxTurns)
		spinner := out.StartSpinner("Thinking...")

		response, err := runner.Send(systemPromptText, conversation)
		spinner.Stop()

		if err != nil {
			finalText = fmt.Sprintf("API error: %v", err)
			exitCode = 1
			break
		}

		newUsage := runner.Usage()
		out.APICall(runner.Provider(), runner.Model(),
			newUsage.InputTokens-usage.InputTokens,
			newUsage.OutputTokens-usage.OutputTokens,
			newUsage.CacheCreationInputTokens-usage.CacheCreationInputTokens,
			newUsage.CacheReadInputTokens-usage.CacheReadInputTokens)

		toolCalls := runner.ExtractToolCalls(response, time.Duration(cfg.TimeoutSeconds)*time.Second)
		if len(toolCalls) > 0 {
			results := executeToolCallsVerbose(shell, toolCalls, cfg.CommandLogFile, out)
			conversation = runner.AppendToolResults(conversation, response, results)

			summary := BuildProgressSummary(cfg.ReportFile)
			conversation = CompactConversation(cfg.Provider, conversation, summary)

			progress := ReadProgress(cfg.ReportFile)
			if progress.Answered > lastAnswered {
				idleTurns = 0
				// Only surface the progress line to the agent when something
				// went wrong. On happy-path runs this line was just context
				// noise; totals are always available in the JSON report.
				if progress.Failed > 0 {
					out.Progress(progress.Answered, progress.VerifiedPassed, progress.Failed)
				}
				lastAnswered = progress.Answered
			} else {
				idleTurns++
			}

			if idleTurns >= cfg.MaxIdleTurns {
				finalText = fmt.Sprintf("Stopped after %d consecutive turns without recording a benchmark step. Check %s for the command trace.",
					idleTurns, cfg.CommandLogFile)
				exitCode = 3
				break
			}
			continue
		}

		finalText = runner.ExtractFinalText(response)
		break
	}

	if finalText == "" {
		finalText = fmt.Sprintf("Stopped after reaching max turns (%d).", cfg.MaxTurns)
		exitCode = 2
	}

	if _, err := os.Stat(cfg.ReportFile); err == nil {
		recordUsage(cfg.ToolsDir, cfg.ReportFile, runner)
		if cfg.Finalize {
			runFinalize(cfg.ReportFile, cfg.Stdout, cfg.Stderr)
		}
	}

	// Benchmark mode (TerseSummary=true) suppresses any remaining agent prose:
	// the harness-generated PrintEndBanner below is the authoritative result
	// for this run. Optimization mode (TerseSummary=false) keeps the agent's
	// prose narrative, which is useful when reading logs to look for ideas
	// about what the agent did well or badly.
	if !cfg.TerseSummary {
		out.FinalText(finalText)
	}
	out.Summary(runner.Usage(), runner.Provider())

	// Harness-generated end-of-run summary: reads the JSON report directly
	// so it always reflects recorded state.
	if _, err := os.Stat(cfg.ReportFile); err == nil {
		PrintEndBanner(cfg.Stdout, cfg.ReportFile)
	}

	return LoopResult{
		ExitCode:  exitCode,
		FinalText: finalText,
		Usage:     runner.Usage(),
	}
}

func clearCommandLog(path string) {
	if path != "" {
		_ = os.WriteFile(path, []byte{}, 0o644)
	}
}

func executeToolCallsVerbose(shell *PersistentShell, calls []ToolCall, logFile string, out *OutputWriter) []ToolExecutionResult {
	var results []ToolExecutionResult
	for _, call := range calls {
		out.ToolCall(call.Command)
		start := time.Now()

		output, exitCode, err := shell.Run(call.Command, time.Duration(call.TimeoutSeconds)*time.Second)
		duration := time.Since(start)

		var result ToolExecutionResult
		if err != nil {
			result = ToolExecutionResult{
				ID:      call.ID,
				IsError: true,
				Content: fmt.Sprintf("$ %s\n[runner_error]\n%s", call.Command, err.Error()),
			}
			appendCommandLog(logFile, call.Command, -1, err.Error())
			out.ToolResult(call.Command, -1, err.Error(), duration)
		} else {
			trimmed := TrimToolOutput(output)
			result = ToolExecutionResult{
				ID:      call.ID,
				IsError: exitCode != 0,
				Content: FormatToolResult(call.Command, exitCode, output),
			}
			appendCommandLog(logFile, call.Command, exitCode, trimmed)
			out.ToolResult(call.Command, exitCode, output, duration)
		}
		results = append(results, result)
	}
	return results
}

func appendCommandLog(path, command string, exitCode int, output string) {
	if path == "" {
		return
	}
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"command":   command,
		"exit_code": exitCode,
		"output":    output,
	}
	data, _ := json.Marshal(entry)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(data, '\n'))
}

func recordUsage(toolsDir, reportFile string, runner Runner) {
	usage := runner.Usage()
	script := filepath.Join(toolsDir, "scripts", "record-run-usage.sh")
	cmd := exec.Command(script, // #nosec G204 -- script path is constructed from known toolsDir
		"--report-file", reportFile,
		"--provider", runner.Provider(),
		"--source", runner.Source(),
		"--request-count", fmt.Sprintf("%d", usage.RequestCount),
		"--input-tokens", fmt.Sprintf("%d", usage.InputTokens),
		"--output-tokens", fmt.Sprintf("%d", usage.OutputTokens),
		"--cache-creation-input-tokens", fmt.Sprintf("%d", usage.CacheCreationInputTokens),
		"--cache-read-input-tokens", fmt.Sprintf("%d", usage.CacheReadInputTokens),
	)
	cmd.Dir = toolsDir
	_ = cmd.Run()
}

// finalizeReport moved to summary.go::runFinalizeReport so stdout/stderr are
// threaded through rather than discarded.
