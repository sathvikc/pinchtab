package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromedp/chromedp"
	"github.com/pinchtab/pinchtab/internal/httpx"
)

type valueResponse struct {
	Ref   string  `json:"ref"`
	Value *string `json:"value"` // null when element has no .value property
}

// HandleGetValue returns the current .value of a form element identified by ref.
//
// @Endpoint GET /value
func (h *Handlers) HandleGetValue(w http.ResponseWriter, r *http.Request) {
	tabID := r.URL.Query().Get("tabId")
	h.recordReadRequest(r, "inspect.value", tabID)

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

	val, err := h.getElementValue(tCtx, resolvedTabID, ref)
	if err != nil {
		httpx.Error(w, 500, err)
		return
	}

	httpx.JSON(w, 200, valueResponse{Ref: ref, Value: val})
}

// HandleTabGetValue returns the .value for a tab identified by path ID.
//
// @Endpoint GET /tabs/{id}/value
func (h *Handlers) HandleTabGetValue(w http.ResponseWriter, r *http.Request) {
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

	h.HandleGetValue(w, req)
}

// getElementValue resolves a ref to a DOM node and returns its .value property.
// Returns a nil pointer when the element has no value property.
func (h *Handlers) getElementValue(ctx context.Context, tabID, ref string) (*string, error) {
	cache := h.Bridge.GetRefCache(tabID)
	if cache == nil {
		return nil, fmt.Errorf("ref not found: %s (no snapshot cache — run /snapshot first)", ref)
	}
	target, ok := cache.Lookup(ref)
	if !ok {
		return nil, fmt.Errorf("ref not found: %s", ref)
	}

	nodeID := target.BackendNodeID
	if nodeID == 0 {
		return nil, fmt.Errorf("element not found in DOM (backendNodeId=%d)", nodeID)
	}

	var result *string
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

		// Step 2: Call .value on the element
		var callResult json.RawMessage
		if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", map[string]any{
			"functionDeclaration": `function() { return this.value !== undefined ? String(this.value) : null; }`,
			"objectId":            resolved.Object.ObjectID,
			"returnByValue":       true,
		}, &callResult); err != nil {
			return fmt.Errorf("get element value: %w", err)
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
			return fmt.Errorf("parse value result: %w", err)
		}
		if callParsed.ExceptionDetails != nil && callParsed.ExceptionDetails.Text != "" {
			return fmt.Errorf("get element value: %s", callParsed.ExceptionDetails.Text)
		}

		// null means the element has no .value property
		if callParsed.Result.Value == nil {
			result = nil
		} else {
			s := fmt.Sprint(callParsed.Result.Value)
			result = &s
		}
		return nil
	}))
	if err != nil {
		return nil, err
	}
	return result, nil
}
