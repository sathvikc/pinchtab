package cdpops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// Headless Chromium can hold synthetic mouseMoved dispatches for about five
// seconds waiting for renderer/compositor ack. Bound the real CDP move and
// fall back to DOM mouse events so hover/move tests and simple automation
// stay responsive.
const mouseMoveDispatchTimeout = 50 * time.Millisecond

func normalizeMouseButton(button string) string {
	switch strings.ToLower(strings.TrimSpace(button)) {
	case "right":
		return "right"
	case "middle":
		return "middle"
	default:
		return "left"
	}
}

func validatePointerCoordinates(x, y float64) error {
	if x < 0 || y < 0 {
		return fmt.Errorf("x/y coordinates must be >= 0")
	}
	return nil
}

func dispatchMouseEvent(ctx context.Context, payload map[string]any) error {
	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", payload, nil)
	}))
}

func dispatchRealMouseMove(ctx context.Context, x, y float64, button input.MouseButton, buttons int64) error {
	stepCtx, cancel := context.WithTimeout(ctx, mouseMoveDispatchTimeout)
	defer cancel()
	return chromedp.Run(stepCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return input.DispatchMouseEvent(input.MouseMoved, x, y).
			WithButton(button).
			WithButtons(buttons).
			Do(ctx)
	}))
}

func dispatchMouseMove(ctx context.Context, x, y float64, button input.MouseButton, buttons int64) error {
	err := dispatchRealMouseMove(ctx, x, y, button, buttons)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return dispatchSyntheticMouseMove(ctx, x, y, button, buttons)
}

func dispatchMouseMoveToNode(ctx context.Context, nodeID int64, x, y float64, button input.MouseButton, buttons int64) error {
	err := dispatchRealMouseMove(ctx, x, y, button, buttons)
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return dispatchSyntheticMouseMoveOnNode(ctx, nodeID, button, buttons)
}

func dispatchSyntheticMouseMove(ctx context.Context, x, y float64, button input.MouseButton, buttons int64) error {
	buttonCode := mouseButtonCode(button)
	expr := fmt.Sprintf(`(function() {
		var cx = %f, cy = %f, button = %d, buttons = %d;
		var target = document.elementFromPoint(cx, cy) || document.documentElement;
		var init = {
			clientX: cx, clientY: cy, screenX: cx, screenY: cy,
			button: button, buttons: buttons,
			bubbles: true, cancelable: true, view: window
		};
		target.dispatchEvent(new MouseEvent('mouseover', init));
		target.dispatchEvent(new MouseEvent('mouseenter', Object.assign({}, init, { bubbles: false })));
		target.dispatchEvent(new MouseEvent('mousemove', init));
	})()`, x, y, buttonCode, buttons)
	return chromedp.Run(ctx, chromedp.Evaluate(expr, nil))
}

func dispatchSyntheticMouseMoveOnNode(ctx context.Context, nodeID int64, button input.MouseButton, buttons int64) error {
	var resolveResult json.RawMessage
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.resolveNode", map[string]any{
			"backendNodeId": nodeID,
		}, &resolveResult)
	})); err != nil {
		return fmt.Errorf("DOM.resolveNode: %w", err)
	}

	var resolved struct {
		Object struct {
			ObjectID string `json:"objectId"`
		} `json:"object"`
	}
	if err := json.Unmarshal(resolveResult, &resolved); err != nil {
		return err
	}
	if strings.TrimSpace(resolved.Object.ObjectID) == "" {
		return fmt.Errorf("element not found in DOM (backendNodeId=%d)", nodeID)
	}

	const fn = `function(button, buttons) {
		var r = this.getBoundingClientRect();
		var cx = r.left + r.width / 2;
		var cy = r.top + r.height / 2;
		var init = {
			clientX: cx, clientY: cy, screenX: cx, screenY: cy,
			button: button, buttons: buttons,
			bubbles: true, cancelable: true, view: window
		};
		this.dispatchEvent(new MouseEvent('mouseover', init));
		this.dispatchEvent(new MouseEvent('mouseenter', Object.assign({}, init, { bubbles: false })));
		this.dispatchEvent(new MouseEvent('mousemove', init));
	}`

	return chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", map[string]any{
			"functionDeclaration": fn,
			"objectId":            resolved.Object.ObjectID,
			"arguments": []map[string]any{
				{"value": mouseButtonCode(button)},
				{"value": buttons},
			},
		}, nil)
	}))
}

func mouseButtonCode(button input.MouseButton) int {
	switch button {
	case input.Middle:
		return 1
	case input.Right:
		return 2
	default:
		return 0
	}
}

func MouseMoveByCoordinate(ctx context.Context, x, y float64) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}
	return dispatchMouseMove(ctx, x, y, input.None, 0)
}

func MouseDownByCoordinate(ctx context.Context, x, y float64, button string) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}
	return dispatchMouseEvent(ctx, map[string]any{
		"type":       "mousePressed",
		"button":     normalizeMouseButton(button),
		"clickCount": 1,
		"x":          x,
		"y":          y,
	})
}

func MouseUpByCoordinate(ctx context.Context, x, y float64, button string) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}
	return dispatchMouseEvent(ctx, map[string]any{
		"type":       "mouseReleased",
		"button":     normalizeMouseButton(button),
		"clickCount": 1,
		"x":          x,
		"y":          y,
	})
}

func MouseWheelByCoordinate(ctx context.Context, x, y float64, deltaX, deltaY int) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}

	// Synthetic Input.dispatchMouseEvent(mouseWheel) in --headless=new no
	// longer reliably fires `wheel` JS listeners and can stall on the
	// compositor ack chain. Dispatch a real WheelEvent at the point under
	// the cursor so listeners run, then scroll the window if no listener
	// called preventDefault().
	expr := fmt.Sprintf(`(function() {
		var dx = %d, dy = %d, cx = %f, cy = %f;
		var target = document.elementFromPoint(cx, cy) || document.documentElement;
		var ev = new WheelEvent('wheel', {
			deltaX: dx, deltaY: dy,
			clientX: cx, clientY: cy,
			bubbles: true, cancelable: true
		});
		if (target.dispatchEvent(ev)) {
			window.scrollBy(dx, dy);
		}
	})()`, deltaX, deltaY, x, y)
	return chromedp.Run(ctx, chromedp.Evaluate(expr, nil))
}

func ClickByCoordinate(ctx context.Context, x, y float64) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}
	if err := MouseDownByCoordinate(ctx, x, y, "left"); err != nil {
		return err
	}
	return MouseUpByCoordinate(ctx, x, y, "left")
}

func ClickByNodeID(ctx context.Context, nodeID int64) error {
	x, y, err := PointerPointForNode(ctx, nodeID, true)
	if err != nil {
		return err
	}

	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.focus", map[string]any{"backendNodeId": nodeID}, nil)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mousePressed",
				"button":     "left",
				"clickCount": 1,
				"x":          x, "y": y,
			}, nil)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mouseReleased",
				"button":     "left",
				"clickCount": 1,
				"x":          x, "y": y,
			}, nil)
		}),
	)
}

func DoubleClickByCoordinate(ctx context.Context, x, y float64) error {
	if err := validatePointerCoordinates(x, y); err != nil {
		return err
	}

	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mousePressed",
				"button":     "left",
				"clickCount": 2,
				"x":          x,
				"y":          y,
			}, nil)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mouseReleased",
				"button":     "left",
				"clickCount": 2,
				"x":          x,
				"y":          y,
			}, nil)
		}),
	)
}

func DoubleClickByNodeID(ctx context.Context, nodeID int64) error {
	x, y, err := PointerPointForNode(ctx, nodeID, true)
	if err != nil {
		return err
	}

	return chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.focus", map[string]any{"backendNodeId": nodeID}, nil)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mousePressed",
				"button":     "left",
				"clickCount": 2,
				"x":          x, "y": y,
			}, nil)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.FromContext(ctx).Target.Execute(ctx, "Input.dispatchMouseEvent", map[string]any{
				"type":       "mouseReleased",
				"button":     "left",
				"clickCount": 2,
				"x":          x, "y": y,
			}, nil)
		}),
	)
}

// DragByNodeID drags an element by (dx, dy) pixels using mousePressed → mouseMoved → mouseReleased.
func DragByNodeID(ctx context.Context, nodeID int64, dx, dy int) error {
	x, y, err := PointerPointForNode(ctx, nodeID, true)
	if err != nil {
		return err
	}

	endX := x + float64(dx)
	endY := y + float64(dy)
	dist := math.Sqrt(float64(dx*dx + dy*dy))
	steps := int(dist / 20)
	if steps < 3 {
		steps = 3
	}
	if steps > 20 {
		steps = 20
	}

	if err := dispatchMouseMove(ctx, x, y, input.None, 0); err != nil {
		return err
	}
	if err := dispatchMouseEvent(ctx, map[string]any{
		"type":       "mousePressed",
		"button":     "left",
		"clickCount": 1,
		"x":          x, "y": y,
	}); err != nil {
		return err
	}
	for i := 1; i <= steps; i++ {
		t := float64(i) / float64(steps)
		mx := x + t*float64(dx)
		my := y + t*float64(dy)
		if err := dispatchMouseMove(ctx, mx, my, input.Left, 1); err != nil {
			return err
		}
	}
	return dispatchMouseEvent(ctx, map[string]any{
		"type":       "mouseReleased",
		"button":     "left",
		"clickCount": 1,
		"x":          endX, "y": endY,
	})
}

func HoverByCoordinate(ctx context.Context, x, y float64) error {
	return MouseMoveByCoordinate(ctx, x, y)
}

func ScrollByCoordinate(ctx context.Context, x, y float64, deltaX, deltaY int) error {
	return MouseWheelByCoordinate(ctx, x, y, deltaX, deltaY)
}

func HoverByNodeID(ctx context.Context, nodeID int64) error {
	x, y, err := PointerPointForNode(ctx, nodeID, true)
	if err != nil {
		return err
	}

	return dispatchMouseMoveToNode(ctx, nodeID, x, y, input.None, 0)
}
