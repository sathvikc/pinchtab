package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chromedp/chromedp"
	"github.com/pinchtab/pinchtab/internal/httpx"
)

type attrResponse struct {
	Ref   string  `json:"ref"`
	Name  string  `json:"name"`
	Value *string `json:"value"` // null when the attribute does not exist
}

// HandleGetAttr returns the value of a specific HTML attribute on an element identified by ref.
//
// @Endpoint GET /attr
func (h *Handlers) HandleGetAttr(w http.ResponseWriter, r *http.Request) {
	tabID := r.URL.Query().Get("tabId")
	h.recordReadRequest(r, "inspect.attr", tabID)

	ref := r.URL.Query().Get("ref")
	if ref == "" {
		httpx.Error(w, 400, fmt.Errorf("ref query parameter is required"))
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		httpx.Error(w, 400, fmt.Errorf("name query parameter is required"))
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

	val, err := h.getElementAttr(tCtx, resolvedTabID, ref, name)
	if err != nil {
		httpx.Error(w, 500, err)
		return
	}

	httpx.JSON(w, 200, attrResponse{Ref: ref, Name: name, Value: val})
}

// HandleTabGetAttr returns the attribute value for a tab identified by path ID.
//
// @Endpoint GET /tabs/{id}/attr
func (h *Handlers) HandleTabGetAttr(w http.ResponseWriter, r *http.Request) {
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

	h.HandleGetAttr(w, req)
}

// getElementAttr resolves a ref to a DOM node and returns the value of the
// named HTML attribute. Returns a nil pointer when the attribute does not exist.
func (h *Handlers) getElementAttr(ctx context.Context, tabID, ref, name string) (*string, error) {
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

		// Step 2: Call getAttribute(name) on the element
		var callResult json.RawMessage
		if err := chromedp.FromContext(ctx).Target.Execute(ctx, "Runtime.callFunctionOn", map[string]any{
			"functionDeclaration": `function(n) { var v = this.getAttribute(n); return v !== null ? v : null; }`,
			"objectId":            resolved.Object.ObjectID,
			"arguments":           []map[string]any{{"value": name}},
			"returnByValue":       true,
		}, &callResult); err != nil {
			return fmt.Errorf("get element attribute: %w", err)
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
			return fmt.Errorf("parse attribute result: %w", err)
		}
		if callParsed.ExceptionDetails != nil && callParsed.ExceptionDetails.Text != "" {
			return fmt.Errorf("get element attribute: %s", callParsed.ExceptionDetails.Text)
		}

		// null means the attribute does not exist on the element
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
