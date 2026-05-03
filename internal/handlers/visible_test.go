package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pinchtab/pinchtab/internal/bridge"
	"github.com/pinchtab/pinchtab/internal/config"
)

func TestHandleGetVisible_MissingRef(t *testing.T) {
	h := New(&mockBridge{}, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/visible", nil)
	w := httptest.NewRecorder()
	h.HandleGetVisible(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ref") {
		t.Fatalf("expected error about ref, got %s", w.Body.String())
	}
}

func TestHandleGetVisible_NoTab(t *testing.T) {
	h := New(&mockBridge{failTab: true}, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/visible?ref=e5", nil)
	w := httptest.NewRecorder()
	h.HandleGetVisible(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleGetVisible_NoSnapshotCache(t *testing.T) {
	mb := &visibleMockBridge{refCache: nil}
	h := New(mb, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/visible?ref=e5", nil)
	w := httptest.NewRecorder()
	h.HandleGetVisible(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "no snapshot cache") {
		t.Fatalf("expected snapshot cache error, got %s", w.Body.String())
	}
}

func TestHandleGetVisible_RefNotFound(t *testing.T) {
	mb := &visibleMockBridge{
		refCache: &bridge.RefCache{
			Refs:    map[string]int64{"e1": 100},
			Targets: map[string]bridge.RefTarget{"e1": {BackendNodeID: 100}},
		},
	}
	h := New(mb, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/visible?ref=e99", nil)
	w := httptest.NewRecorder()
	h.HandleGetVisible(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "ref not found: e99") {
		t.Fatalf("expected ref not found error, got %s", w.Body.String())
	}
}

func TestHandleTabGetVisible_MissingTabID(t *testing.T) {
	h := New(&mockBridge{}, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/tabs//visible?ref=e5", nil)
	w := httptest.NewRecorder()
	h.HandleTabGetVisible(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleTabGetVisible_ForwardsTabID(t *testing.T) {
	h := New(&mockBridge{failTab: true}, &config.RuntimeConfig{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/tabs/tab_abc/visible?ref=e5", nil)
	req.SetPathValue("id", "tab_abc")
	w := httptest.NewRecorder()
	h.HandleTabGetVisible(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestVisibleRoutesRegistered(t *testing.T) {
	h := New(&mockBridge{failTab: true}, &config.RuntimeConfig{}, nil, nil, nil)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux, nil)

	paths := []string{"/visible?ref=e1", "/tabs/tab1/visible?ref=e1"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)
			if w.Code == http.StatusNotFound && strings.Contains(w.Body.String(), "404 page not found") {
				t.Fatalf("route %s not registered", path)
			}
		})
	}
}

// visibleMockBridge extends mockBridge with GetRefCache support.
type visibleMockBridge struct {
	mockBridge
	refCache *bridge.RefCache
}

func (m *visibleMockBridge) GetRefCache(tabID string) *bridge.RefCache {
	return m.refCache
}
