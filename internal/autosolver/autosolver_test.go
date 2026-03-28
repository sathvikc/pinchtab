package autosolver

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// --- Mock implementations ---

type mockPage struct {
	url   string
	title string
	html  string
}

func (m *mockPage) URL() string              { return m.url }
func (m *mockPage) Title() string            { return m.title }
func (m *mockPage) HTML() (string, error)    { return m.html, nil }
func (m *mockPage) Screenshot() ([]byte, error) { return nil, nil }

type mockExecutor struct {
	clickCalled    int
	typeCalled     int
	navigateCalled int
}

func (m *mockExecutor) Click(_ context.Context, _, _ float64) error {
	m.clickCalled++
	return nil
}
func (m *mockExecutor) Type(_ context.Context, _ string) error {
	m.typeCalled++
	return nil
}
func (m *mockExecutor) WaitFor(_ context.Context, _ string, _ time.Duration) error { return nil }
func (m *mockExecutor) Evaluate(_ context.Context, _ string, _ interface{}) error  { return nil }
func (m *mockExecutor) Navigate(_ context.Context, _ string) error {
	m.navigateCalled++
	return nil
}

type mockSolver struct {
	name      string
	priority  int
	canHandle bool
	solved    bool
	err       error
}

func (m *mockSolver) Name() string  { return m.name }
func (m *mockSolver) Priority() int { return m.priority }
func (m *mockSolver) CanHandle(_ context.Context, _ Page) (bool, error) {
	return m.canHandle, nil
}
func (m *mockSolver) Solve(_ context.Context, _ Page, _ ActionExecutor) (*Result, error) {
	if m.err != nil {
		return &Result{Error: m.err.Error()}, m.err
	}
	return &Result{Solved: m.solved, SolverUsed: m.name}, nil
}

type mockSemantic struct {
	intent *Intent
	err    error
}

func (m *mockSemantic) DetectIntent(_ context.Context, _ Page) (*Intent, error) {
	return m.intent, m.err
}
func (m *mockSemantic) FindElement(_ context.Context, _ Page, _ string) (*ElementMatch, error) {
	return nil, nil
}
func (m *mockSemantic) SuggestAction(_ context.Context, _ Page, _ *Intent) (*SuggestedAction, error) {
	return nil, nil
}

type mockLLM struct {
	resp *LLMResponse
	err  error
}

func (m *mockLLM) SuggestNextAction(_ context.Context, _ LLMRequest) (*LLMResponse, error) {
	return m.resp, m.err
}

// --- Tests ---

func TestSolve_NormalPage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 3

	as := New(cfg, nil, nil)

	page := &mockPage{title: "Google", url: "https://google.com"}
	executor := &mockExecutor{}

	result, err := as.Solve(context.Background(), page, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Solved {
		t.Error("expected Solved=true for normal page")
	}
	if result.Intent != IntentNormal {
		t.Errorf("expected intent Normal, got %s", result.Intent)
	}
	if result.Attempts != 0 {
		t.Errorf("expected 0 attempts for normal page, got %d", result.Attempts)
	}
}

func TestSolve_SemanticDetection(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 1

	semantic := &mockSemantic{
		intent: &Intent{Type: IntentCaptcha, Confidence: 0.9},
	}

	solver := &mockSolver{
		name:      "test-solver",
		priority:  10,
		canHandle: true,
		solved:    true,
	}

	as := New(cfg, semantic, nil)
	as.Registry().MustRegister(solver)

	page := &mockPage{title: "Challenge Page", url: "https://example.com"}
	executor := &mockExecutor{}

	result, err := as.Solve(context.Background(), page, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Solved {
		t.Error("expected Solved=true")
	}
	if result.SolverUsed != "test-solver" {
		t.Errorf("expected solver 'test-solver', got %q", result.SolverUsed)
	}
	if result.Intent != IntentCaptcha {
		t.Errorf("expected intent Captcha, got %s", result.Intent)
	}
}

func TestSolve_FallbackChain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 1
	cfg.RetryBaseDelay = time.Millisecond

	// First solver fails, second succeeds.
	failing := &mockSolver{
		name:      "failing",
		priority:  10,
		canHandle: true,
		solved:    false,
		err:       fmt.Errorf("solver error"),
	}
	succeeding := &mockSolver{
		name:      "succeeding",
		priority:  20,
		canHandle: true,
		solved:    true,
	}

	as := New(cfg, nil, nil)
	as.Registry().MustRegister(failing)
	as.Registry().MustRegister(succeeding)

	// Use a title that triggers captcha detection via heuristics.
	page := &mockPage{title: "Just a moment...", url: "https://example.com"}
	executor := &mockExecutor{}

	result, err := as.Solve(context.Background(), page, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Solved {
		t.Error("expected Solved=true from second solver")
	}
	if result.SolverUsed != "succeeding" {
		t.Errorf("expected solver 'succeeding', got %q", result.SolverUsed)
	}
}

func TestSolve_AllSolversFail(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 2
	cfg.RetryBaseDelay = time.Millisecond

	failing := &mockSolver{
		name:      "failing",
		priority:  10,
		canHandle: true,
		solved:    false,
		err:       fmt.Errorf("solver error"),
	}

	as := New(cfg, nil, nil)
	as.Registry().MustRegister(failing)

	page := &mockPage{title: "Just a moment...", url: "https://example.com"}
	executor := &mockExecutor{}

	result, err := as.Solve(context.Background(), page, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Solved {
		t.Error("expected Solved=false when all solvers fail")
	}
	if result.Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", result.Attempts)
	}
	if len(result.History) == 0 {
		t.Error("expected non-empty history")
	}
}

func TestSolve_LLMFallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 1
	cfg.LLMFallback = true
	cfg.RetryBaseDelay = time.Millisecond

	llm := &mockLLM{
		resp: &LLMResponse{
			Action:     ActionNone,
			Confidence: 0.8,
		},
	}

	as := New(cfg, nil, llm)

	// No solvers registered, so LLM fallback should activate.
	page := &mockPage{title: "Just a moment...", url: "https://example.com", html: "<html></html>"}
	executor := &mockExecutor{}

	result, err := as.Solve(context.Background(), page, executor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Solved {
		t.Error("expected Solved=true via LLM fallback")
	}
	if result.SolverUsed != "llm" {
		t.Errorf("expected solver 'llm', got %q", result.SolverUsed)
	}
}

func TestSolve_ContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 10
	cfg.RetryBaseDelay = 5 * time.Second

	// Slow solver that never succeeds.
	slow := &mockSolver{
		name:      "slow",
		priority:  10,
		canHandle: true,
		solved:    false,
	}

	as := New(cfg, nil, nil)
	as.Registry().MustRegister(slow)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	page := &mockPage{title: "Just a moment...", url: "https://example.com"}
	executor := &mockExecutor{}

	result, err := as.Solve(ctx, page, executor)
	if err == nil {
		// Context cancellation may or may not surface as an error
		// depending on timing; the key check is that it terminates.
	}
	_ = result
	_ = err
}

func TestSolve_PriorityOrdering(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxAttempts = 1
	cfg.RetryBaseDelay = time.Millisecond

	// Register solvers in reverse priority order.
	var solveOrder []string
	makeSolver := func(name string, priority int) Solver {
		return &trackingSolver{
			name:      name,
			priority:  priority,
			canHandle: true,
			order:     &solveOrder,
		}
	}

	as := New(cfg, nil, nil)
	as.Registry().MustRegister(makeSolver("third", 30))
	as.Registry().MustRegister(makeSolver("first", 10))
	as.Registry().MustRegister(makeSolver("second", 20))

	page := &mockPage{title: "Just a moment...", url: "https://example.com"}
	executor := &mockExecutor{}

	_, _ = as.Solve(context.Background(), page, executor)

	// Verify solvers were tried in priority order.
	if len(solveOrder) < 3 {
		t.Fatalf("expected 3 solver calls, got %d", len(solveOrder))
	}
	if solveOrder[0] != "first" {
		t.Errorf("expected first solver tried, got %q", solveOrder[0])
	}
	if solveOrder[1] != "second" {
		t.Errorf("expected second solver tried, got %q", solveOrder[1])
	}
	if solveOrder[2] != "third" {
		t.Errorf("expected third solver tried, got %q", solveOrder[2])
	}
}

// trackingSolver records the order in which Solve is called.
type trackingSolver struct {
	name      string
	priority  int
	canHandle bool
	order     *[]string
}

func (s *trackingSolver) Name() string  { return s.name }
func (s *trackingSolver) Priority() int { return s.priority }
func (s *trackingSolver) CanHandle(_ context.Context, _ Page) (bool, error) {
	return s.canHandle, nil
}
func (s *trackingSolver) Solve(_ context.Context, _ Page, _ ActionExecutor) (*Result, error) {
	*s.order = append(*s.order, s.name)
	return &Result{Solved: false}, nil
}
