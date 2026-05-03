package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromedp/chromedp"
	"github.com/pinchtab/pinchtab/internal/httpx"
)

type checkedResponse struct {
	Ref     string `json:"ref"`
	Checked bool   `json:"checked"`
}

// HandleGetChecked returns whether an element identified by ref is checked.
//
// @Endpoint GET /checked
func (h *Handlers) HandleGetChecked(w http.ResponseWriter, r *http.Request) {
	tabID := r.URL.Query().Get("tabId")
	h.recordReadRequest(r, "inspect.checked", tabID)

	ref := r.URL.Query().Get("ref")
	if ref == "" {
		httpx.Error(w, 400, fmt.Errorf("ref query parameter is required"))
		return
	}

	if err := h.ensureChrome(); err != nil {
		if h.writeBridgeUnavailable(w, err) {
			return
		}
		httpx.Error(w, 500, fmt.Errorf("chrome initialization: %w", err))
		return
	}

	ctx, resolvedTabID, err := h.tabContextWithHeader(w, r, tabID)
	if err != nil {
		WriteTabContextError(w, err, 404)
		return
	}
	if _, ok := h.enforceCurrentTabDomainPolicy(w, r, ctx, resolvedTabID); !ok {
		return
	}
	defer h.armAutoCloseIfEnabled(resolvedTabID)

	tCtx, tCancel := context.WithTimeout(ctx, h.Config.ActionTimeout)
	defer tCancel()
	go httpx.CancelOnClientDone(r.Context(), tCancel)

	checked, err := h.getElementChecked(tCtx, resolvedTabID, ref)
	if err != nil {
		httpx.Error(w, 500, err)
		return
	}

	httpx.JSON(w, 200, checkedResponse{Ref: ref, Checked: checked})
}

// HandleTabGetChecked returns checked state for a tab identified by path ID.
//
// @Endpoint GET /tabs/{id}/checked
func (h *Handlers) HandleTabGetChecked(w http.ResponseWriter, r *http.Request) {
	tabID := r.PathValue("id")
	if tabID == "" {
		httpx.Error(w, 400, fmt.Errorf("tab id required"))
		return
	}

	q := r.URL.Query()
	q.Set("tabId", tabID)

	req := r.Clone(r.Context())
	u := *r.URL
	u.RawQuery = q.Encode()
	req.URL = &u

	h.HandleGetChecked(w, req)
}

// getElementChecked resolves a ref to a DOM node and checks whether it is checked.
func (h *Handlers) getElementChecked(ctx context.Context, tabID, ref string) (bool, error) {
	cache := h.Bridge.GetRefCache(tabID)
	if cache == nil {
		return false, fmt.Errorf("ref not found: %s (no snapshot cache — run /snapshot first)", ref)
	}
	target, ok := cache.Lookup(ref)
	if !ok {
		return false, fmt.Errorf("ref not found: %s", ref)
	}

	nodeID := target.BackendNodeID
	if nodeID == 0 {
		return false, fmt.Errorf("element not found in DOM (backendNodeId=%d)", nodeID)
	}

	var checked bool
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// Step 1: Resolve backend node ID to a remote object
		var resolveResult json.RawMessage
		if err := chromedp.FromContext(ctx).Target.Execute(ctx, "DOM.resolveNode", map[string]any{
			"backendNodeId": nodeID,
		}, &resolveResult); err != nil {
			return fmt.Errorf("resolve node: %w", err)
		}

		var resolved struct {
			Object struct {
				ObjectID string `json:"objectId"`
			} `json:"object"`
		}
		if err := json.Unmarshal(resolveResult, &resolved); err != nil {
			return fmt.Errorf("parse resolved node: %w", err)
		}
		if resolved.Object.ObjectID == "" {
			return fmt.Errorf("element not found in DOM (backendNodeId=%d)", nodeID)
		}

		// Step 2: Evaluate checked check on the element
		var callResult json.RawMessage
		if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", map[string]any{
			"functionDeclaration": `function() { return !!this.checked; }`,
			"objectId":            resolved.Object.ObjectID,
			"returnByValue":       true,
		}, &callResult); err != nil {
			return fmt.Errorf("check element checked state: %w", err)
		}

		var callParsed struct {
			Result struct {
				Type  string `json:"type"`
				Value any    `json:"value"`
			} `json:"result"`
			ExceptionDetails *struct {
				Text string `json:"text"`
			} `json:"exceptionDetails,omitempty"`
		}
		if err := json.Unmarshal(callResult, &callParsed); err != nil {
			return fmt.Errorf("parse checked result: %w", err)
		}
		if callParsed.ExceptionDetails != nil && callParsed.ExceptionDetails.Text != "" {
			return fmt.Errorf("check element checked state: %s", callParsed.ExceptionDetails.Text)
		}

		if b, ok := callParsed.Result.Value.(bool); ok {
			checked = b
		}
		return nil
	}))
	if err != nil {
		return false, err
	}
	return checked, nil
}
