package cdpops

import (
	"context"
	"errors"
	"testing"

	"github.com/chromedp/cdproto/input"
)

func TestDispatchMouseMoveFallsBackToSyntheticOnDeadline(t *testing.T) {
	origReal := dispatchRealMouseMoveFunc
	origSynthetic := dispatchSyntheticMouseMoveFunc
	t.Cleanup(func() {
		dispatchRealMouseMoveFunc = origReal
		dispatchSyntheticMouseMoveFunc = origSynthetic
	})

	dispatchRealMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		return context.DeadlineExceeded
	}

	called := false
	dispatchSyntheticMouseMoveFunc = func(_ context.Context, x, y float64, button input.MouseButton, buttons int64) error {
		called = true
		if x != 12 || y != 34 {
			t.Fatalf("synthetic move coordinates = (%v, %v), want (12, 34)", x, y)
		}
		if button != input.Left || buttons != 1 {
			t.Fatalf("synthetic move button state = (%v, %d), want (%v, 1)", button, buttons, input.Left)
		}
		return nil
	}

	if err := dispatchMouseMove(context.Background(), 12, 34, input.Left, 1); err != nil {
		t.Fatalf("dispatchMouseMove returned error: %v", err)
	}
	if !called {
		t.Fatal("expected synthetic fallback to run")
	}
}

func TestDispatchMouseMoveDoesNotFallbackOnNonDeadlineError(t *testing.T) {
	origReal := dispatchRealMouseMoveFunc
	origSynthetic := dispatchSyntheticMouseMoveFunc
	t.Cleanup(func() {
		dispatchRealMouseMoveFunc = origReal
		dispatchSyntheticMouseMoveFunc = origSynthetic
	})

	want := errors.New("cdp failed")
	dispatchRealMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		return want
	}
	dispatchSyntheticMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		t.Fatal("synthetic fallback should not run for non-timeout CDP errors")
		return nil
	}

	if err := dispatchMouseMove(context.Background(), 12, 34, input.None, 0); !errors.Is(err, want) {
		t.Fatalf("dispatchMouseMove error = %v, want %v", err, want)
	}
}

func TestDispatchMouseMoveContextCancellationWinsOverFallback(t *testing.T) {
	origReal := dispatchRealMouseMoveFunc
	origSynthetic := dispatchSyntheticMouseMoveFunc
	t.Cleanup(func() {
		dispatchRealMouseMoveFunc = origReal
		dispatchSyntheticMouseMoveFunc = origSynthetic
	})

	dispatchRealMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		return context.DeadlineExceeded
	}
	dispatchSyntheticMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		t.Fatal("synthetic fallback should not run after caller context cancellation")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := dispatchMouseMove(ctx, 12, 34, input.None, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("dispatchMouseMove error = %v, want context.Canceled", err)
	}
}

func TestDispatchMouseMoveToNodeFallsBackToSyntheticNodeMove(t *testing.T) {
	origReal := dispatchRealMouseMoveFunc
	origSyntheticNode := dispatchSyntheticMouseMoveOnNodeFunc
	t.Cleanup(func() {
		dispatchRealMouseMoveFunc = origReal
		dispatchSyntheticMouseMoveOnNodeFunc = origSyntheticNode
	})

	dispatchRealMouseMoveFunc = func(context.Context, float64, float64, input.MouseButton, int64) error {
		return context.DeadlineExceeded
	}

	called := false
	dispatchSyntheticMouseMoveOnNodeFunc = func(_ context.Context, nodeID int64, button input.MouseButton, buttons int64) error {
		called = true
		if nodeID != 42 {
			t.Fatalf("nodeID = %d, want 42", nodeID)
		}
		if button != input.Right || buttons != 2 {
			t.Fatalf("button state = (%v, %d), want (%v, 2)", button, buttons, input.Right)
		}
		return nil
	}

	if err := dispatchMouseMoveToNode(context.Background(), 42, 12, 34, input.Right, 2); err != nil {
		t.Fatalf("dispatchMouseMoveToNode returned error: %v", err)
	}
	if !called {
		t.Fatal("expected synthetic node fallback to run")
	}
}
