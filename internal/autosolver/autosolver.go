package autosolver

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// AutoSolver orchestrates the challenge-detection and solving pipeline.
// It uses a fallback chain: built-in solvers → semantic engine → external
// solvers → LLM provider, trying each layer before moving to the next.
type AutoSolver struct {
	registry *Registry
	semantic SemanticEngine
	llm      LLMProvider
	config   Config
}

// New creates an AutoSolver with the given configuration.
// The semantic engine and LLM provider are optional (can be nil).
func New(cfg Config, semantic SemanticEngine, llm LLMProvider) *AutoSolver {
	return &AutoSolver{
		registry: NewRegistry(),
		semantic: semantic,
		llm:      llm,
		config:   cfg,
	}
}

// Registry returns the solver registry for external registration.
func (as *AutoSolver) Registry() *Registry {
	return as.registry
}

// Solve runs the autosolver pipeline on the current page.
//
// Steps:
//  1. Detect intent via semantic engine (or title-based heuristics)
//  2. If no challenge detected, return immediately
//  3. Try matching solvers in priority order
//  4. If all fail and LLM is enabled, try LLM fallback
//  5. Return result with full attempt history
func (as *AutoSolver) Solve(ctx context.Context, page Page, executor ActionExecutor) (*Result, error) {
	start := time.Now()
	result := &Result{
		FinalTitle: page.Title(),
		FinalURL:   page.URL(),
	}

	slog.Info("autosolver_start",
		"url", page.URL(),
		"title", page.Title(),
		"max_attempts", as.config.MaxAttempts,
		"llm_fallback", as.config.LLMFallback)

	// Detect what kind of page we're dealing with.
	intent, err := as.detectIntent(ctx, page)
	if err != nil {
		slog.Warn("autosolver: intent detection failed, proceeding with unknown",
			"err", err, "url", page.URL())
		intent = &Intent{Type: IntentUnknown, Confidence: 0}
	}
	result.Intent = intent.Type

	// No challenge — nothing to solve.
	if intent.Type == IntentNormal {
		result.Solved = true
		result.TotalDuration = time.Since(start)
		slog.Info("autosolver_done",
			"solved", true,
			"reason", "no_challenge_detected",
			"url", page.URL(),
			"duration_ms", result.TotalDuration.Milliseconds())
		return result, nil
	}

	slog.Info("autosolver: challenge detected",
		"type", intent.Type,
		"confidence", intent.Confidence,
		"url", page.URL())

	// Run the fallback chain with retry logic.
	for attempt := 0; attempt < as.config.MaxAttempts; attempt++ {
		result.Attempts = attempt + 1

		// Apply backoff between retries (skip first attempt).
		if attempt > 0 {
			delay := as.backoffDelay(attempt)
			slog.Info("autosolver_retry",
				"attempt", attempt+1,
				"delay_ms", delay.Milliseconds(),
				"url", page.URL())
			select {
			case <-ctx.Done():
				result.TotalDuration = time.Since(start)
				result.Error = ctx.Err().Error()
				slog.Warn("autosolver_done",
					"solved", false,
					"reason", "context_cancelled",
					"attempts", result.Attempts,
					"duration_ms", result.TotalDuration.Milliseconds())
				return result, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Try registered solvers in priority order.
		solved, entry := as.trySolvers(ctx, page, executor)
		if entry != nil {
			result.History = append(result.History, *entry)
		}
		if solved {
			result.Solved = true
			result.SolverUsed = entry.Solver
			result.FinalTitle = page.Title()
			result.FinalURL = page.URL()
			result.TotalDuration = time.Since(start)
			slog.Info("autosolver_success",
				"solver", entry.Solver,
				"attempts", result.Attempts,
				"duration_ms", result.TotalDuration.Milliseconds(),
				"url", page.URL())
			slog.Info("autosolver_done",
				"solved", true,
				"solver", entry.Solver,
				"attempts", result.Attempts,
				"duration_ms", result.TotalDuration.Milliseconds())
			return result, nil
		}

		// Try LLM fallback if enabled and all solvers failed.
		if as.config.LLMFallback && as.llm != nil {
			solved, entry := as.tryLLM(ctx, page, executor, result.History)
			if entry != nil {
				result.History = append(result.History, *entry)
			}
			if solved {
				result.Solved = true
				result.SolverUsed = "llm"
				result.FinalTitle = page.Title()
				result.FinalURL = page.URL()
				result.TotalDuration = time.Since(start)
				slog.Info("autosolver_success",
					"solver", "llm",
					"attempts", result.Attempts,
					"duration_ms", result.TotalDuration.Milliseconds(),
					"url", page.URL())
				slog.Info("autosolver_done",
					"solved", true,
					"solver", "llm",
					"attempts", result.Attempts,
					"duration_ms", result.TotalDuration.Milliseconds())
				return result, nil
			}
		}
	}

	result.TotalDuration = time.Since(start)
	result.Error = fmt.Sprintf("all %d attempts exhausted", as.config.MaxAttempts)
	slog.Warn("autosolver_failure",
		"attempts", result.Attempts,
		"duration_ms", result.TotalDuration.Milliseconds(),
		"url", page.URL(),
		"error", result.Error)
	slog.Info("autosolver_done",
		"solved", false,
		"reason", "max_attempts_exhausted",
		"attempts", result.Attempts,
		"duration_ms", result.TotalDuration.Milliseconds())
	return result, nil
}

// detectIntent uses the semantic engine if available, otherwise falls
// back to basic title-based heuristics.
func (as *AutoSolver) detectIntent(ctx context.Context, page Page) (*Intent, error) {
	if as.semantic != nil {
		return as.semantic.DetectIntent(ctx, page)
	}
	return detectIntentByTitle(page.Title()), nil
}

// trySolvers iterates through matching solvers and returns on first success.
func (as *AutoSolver) trySolvers(ctx context.Context, page Page, executor ActionExecutor) (bool, *AttemptEntry) {
	solvers := as.registry.MatchingSolvers(ctx, page)
	if len(solvers) == 0 {
		return false, &AttemptEntry{
			Solver: "none",
			Status: StatusSkipped,
		}
	}

	for _, s := range solvers {
		solverCtx, cancel := context.WithTimeout(ctx, as.config.SolverTimeout)
		solverStart := time.Now()

		slog.Info("autosolver_attempt",
			"solver", s.Name(),
			"priority", s.Priority())

		solveResult, err := s.Solve(solverCtx, page, executor)
		cancel()

		entry := &AttemptEntry{
			Solver:   s.Name(),
			Duration: time.Since(solverStart),
		}

		if err != nil {
			entry.Status = StatusFailed
			entry.Error = err.Error()
			slog.Warn("autosolver_failure",
				"solver", s.Name(),
				"error", err,
				"duration_ms", entry.Duration.Milliseconds())
			continue
		}

		if solveResult != nil && solveResult.Solved {
			entry.Status = StatusSolved
			return true, entry
		}

		entry.Status = StatusFailed
		if solveResult != nil && solveResult.Error != "" {
			entry.Error = solveResult.Error
		}
		slog.Debug("autosolver: solver returned not-solved",
			"solver", s.Name(),
			"duration_ms", entry.Duration.Milliseconds())
	}

	return false, &AttemptEntry{
		Solver: solvers[len(solvers)-1].Name(),
		Status: StatusFailed,
		Error:  "all matching solvers failed",
	}
}

// tryLLM builds a trimmed request and asks the LLM for the next action.
func (as *AutoSolver) tryLLM(ctx context.Context, page Page, executor ActionExecutor, history []AttemptEntry) (bool, *AttemptEntry) {
	llmStart := time.Now()
	entry := &AttemptEntry{Solver: "llm"}

	html, err := page.HTML()
	if err != nil {
		entry.Status = StatusFailed
		entry.Error = fmt.Sprintf("get HTML: %v", err)
		entry.Duration = time.Since(llmStart)
		return false, entry
	}

	// Trim HTML to reduce token usage (max ~4000 chars).
	if len(html) > 4000 {
		html = html[:4000]
	}

	resp, err := as.llm.SuggestNextAction(ctx, LLMRequest{
		PageTitle:    page.Title(),
		PageURL:      page.URL(),
		TrimmedHTML:  html,
		DetectedType: IntentUnknown,
		PrevAttempts: history,
	})
	if err != nil {
		entry.Status = StatusFailed
		entry.Error = fmt.Sprintf("llm: %v", err)
		entry.Duration = time.Since(llmStart)
		return false, entry
	}

	// Execute the LLM's suggested action.
	if err := executeAction(ctx, executor, resp); err != nil {
		entry.Status = StatusFailed
		entry.Error = fmt.Sprintf("execute llm action: %v", err)
		entry.Duration = time.Since(llmStart)
		return false, entry
	}

	entry.Status = StatusSolved
	entry.Duration = time.Since(llmStart)
	return true, entry
}

// executeAction translates an LLMResponse into an ActionExecutor call.
func executeAction(ctx context.Context, executor ActionExecutor, resp *LLMResponse) error {
	if resp == nil {
		return fmt.Errorf("nil response")
	}

	switch resp.Action {
	case ActionClick:
		if resp.Selector != "" {
			// Resolve selector to coordinates via evaluate.
			var coords struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			}
			expr := fmt.Sprintf(`(() => {
				const el = document.querySelector(%q);
				if (!el) return null;
				const r = el.getBoundingClientRect();
				return {x: r.x + r.width/2, y: r.y + r.height/2};
			})()`, resp.Selector)
			if err := executor.Evaluate(ctx, expr, &coords); err != nil {
				return fmt.Errorf("resolve selector %q: %w", resp.Selector, err)
			}
			return executor.Click(ctx, coords.X, coords.Y)
		}
		return fmt.Errorf("click action requires selector")

	case ActionType_:
		return executor.Type(ctx, resp.Text)

	case ActionNavigate:
		return executor.Navigate(ctx, resp.URL)

	case ActionNone:
		return nil

	default:
		return fmt.Errorf("unsupported action: %s", resp.Action)
	}
}

// backoffDelay calculates exponential backoff with jitter.
func (as *AutoSolver) backoffDelay(attempt int) time.Duration {
	base := as.config.RetryBaseDelay
	if base <= 0 {
		base = 500 * time.Millisecond
	}
	maxDelay := as.config.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 10 * time.Second
	}

	delay := base * time.Duration(1<<uint(attempt-1))
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}
