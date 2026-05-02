package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/chromedp"
	"github.com/pinchtab/pinchtab/internal/activity"
	"github.com/pinchtab/pinchtab/internal/httpx"
)

// credentialPair holds HTTP basic auth credentials for a tab.
type credentialPair struct {
	Username string
	Password string
}

// credentialStore provides thread-safe per-tab credential storage.
type credentialStore struct {
	mu          sync.RWMutex
	credentials map[string]*credentialPair
}

func newCredentialStore() *credentialStore {
	return &credentialStore{
		credentials: make(map[string]*credentialPair),
	}
}

func (cs *credentialStore) Set(tabID string, cred *credentialPair) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.credentials[tabID] = cred
}

func (cs *credentialStore) Get(tabID string) (*credentialPair, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	cred, ok := cs.credentials[tabID]
	return cred, ok
}

func (cs *credentialStore) Delete(tabID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.credentials, tabID)
}

type credentialsRequest struct {
	TabID    string  `json:"tabId"`
	Username *string `json:"username"`
	Password string  `json:"password"`
}

// HandleSetCredentials sets HTTP auth credentials via CDP Fetch domain.
// POST /emulation/credentials
func (h *Handlers) HandleSetCredentials(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize)).Decode(&req); err != nil {
		httpx.Error(w, 400, fmt.Errorf("decode: %w", err))
		return
	}

	h.setCredentials(w, r, req)
}

// HandleTabSetCredentials sets HTTP auth credentials for a specific tab.
// POST /tabs/{id}/emulation/credentials
func (h *Handlers) HandleTabSetCredentials(w http.ResponseWriter, r *http.Request) {
	tabID := r.PathValue("id")
	if tabID == "" {
		httpx.Error(w, 400, fmt.Errorf("missing tab ID"))
		return
	}

	var req credentialsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodySize)).Decode(&req); err != nil {
		httpx.Error(w, 400, fmt.Errorf("decode: %w", err))
		return
	}

	if req.TabID != "" && req.TabID != tabID {
		httpx.Error(w, 400, fmt.Errorf("tabId in body %q does not match URL path %q", req.TabID, tabID))
		return
	}
	req.TabID = tabID

	h.setCredentials(w, r, req)
}

func (h *Handlers) setCredentials(w http.ResponseWriter, r *http.Request, req credentialsRequest) {
	if req.Username == nil {
		httpx.Error(w, 400, fmt.Errorf("missing required field: username"))
		return
	}

	username := *req.Username

	// Non-empty username requires password field (empty password is allowed).
	// Empty username means "clear credentials".

	ctx, resolvedTabID, err := h.tabContext(r, req.TabID)
	if err != nil {
		WriteTabContextError(w, err, 404)
		return
	}
	if _, ok := h.enforceCurrentTabDomainPolicy(w, r, ctx, resolvedTabID); !ok {
		return
	}

	tCtx, tCancel := context.WithTimeout(ctx, 5*time.Second)
	defer tCancel()

	if username == "" {
		// Clear credentials: disable fetch domain and remove stored credentials.
		h.credentialStore.Delete(resolvedTabID)

		if err := chromedp.Run(tCtx,
			chromedp.ActionFunc(func(ctx context.Context) error {
				return fetch.Disable().Do(ctx)
			}),
		); err != nil {
			httpx.Error(w, 500, fmt.Errorf("CDP fetch disable: %w", err))
			return
		}

		h.recordActivity(r, activity.Update{Action: "emulation.credentials", TabID: resolvedTabID})

		httpx.JSON(w, 200, map[string]any{
			"status": "cleared",
		})
		return
	}

	// Store credentials for the event listener to reference.
	h.credentialStore.Set(resolvedTabID, &credentialPair{
		Username: username,
		Password: req.Password,
	})

	// Enable Fetch domain with auth request interception.
	if err := chromedp.Run(tCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return fetch.Enable().WithHandleAuthRequests(true).Do(ctx)
		}),
	); err != nil {
		httpx.Error(w, 500, fmt.Errorf("CDP fetch enable: %w", err))
		return
	}

	// Install event listener for auth challenges and paused requests.
	// Use the non-timeout tab context so the listener persists beyond this request.
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *fetch.EventAuthRequired:
			go func() {
				cred, ok := h.credentialStore.Get(resolvedTabID)
				if !ok {
					return
				}
				_ = chromedp.Run(ctx, chromedp.ActionFunc(func(innerCtx context.Context) error {
					return fetch.ContinueWithAuth(e.RequestID, &fetch.AuthChallengeResponse{
						Response: fetch.AuthChallengeResponseResponseProvideCredentials,
						Username: cred.Username,
						Password: cred.Password,
					}).Do(innerCtx)
				}))
			}()
		case *fetch.EventRequestPaused:
			go func() {
				_ = chromedp.Run(ctx, chromedp.ActionFunc(func(innerCtx context.Context) error {
					return fetch.ContinueRequest(e.RequestID).Do(innerCtx)
				}))
			}()
		}
	})

	h.recordActivity(r, activity.Update{Action: "emulation.credentials", TabID: resolvedTabID})

	httpx.JSON(w, 200, map[string]any{
		"username": username,
		"status":   "applied",
	})
}
