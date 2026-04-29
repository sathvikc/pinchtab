package actions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newNavigateCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("new-tab", false, "")
	cmd.Flags().Bool("block-images", false, "")
	cmd.Flags().Bool("block-ads", false, "")
	cmd.Flags().String("tab", "", "")
	cmd.Flags().Bool("print-tab-id", false, "")
	return cmd
}

func TestNavigate(t *testing.T) {
	m := newMockServer()
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	Navigate(client, m.base(), "", "https://pinchtab.com", cmd)
	if m.lastMethod != "POST" {
		t.Errorf("expected POST, got %s", m.lastMethod)
	}
	if m.lastPath != "/navigate" {
		t.Errorf("expected /navigate, got %s", m.lastPath)
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(m.lastBody), &body)
	if body["url"] != "https://pinchtab.com" {
		t.Errorf("expected url=https://pinchtab.com, got %v", body["url"])
	}
}

func TestNavigateReusesImplicitTabWhenItExists(t *testing.T) {
	m := newMockServer()
	m.response = `{"tabs":[{"id":"ABC123"}],"tabId":"ABC123","status":"ok"}`
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	cmd.Flags().Lookup("tab").DefValue = "ABC123"
	_ = cmd.Flags().Set("tab", "ABC123")
	cmd.Flags().Lookup("tab").Changed = false

	Navigate(client, m.base(), "", "https://pinchtab.com", cmd)

	if len(m.requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(m.requests))
	}
	if m.requests[0].Path != "/tabs" {
		t.Fatalf("first request path = %q, want /tabs", m.requests[0].Path)
	}
	if m.requests[1].Path != "/tabs/ABC123/navigate" {
		t.Fatalf("navigate path = %q, want /tabs/ABC123/navigate", m.requests[1].Path)
	}
}

func TestNavigateFallsBackToNewTabForStaleImplicitTab(t *testing.T) {
	m := newMockServer()
	m.response = `{"tabs":[],"tabId":"NEW123","status":"ok"}`
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	cmd.Flags().Lookup("tab").DefValue = "STALE123"
	_ = cmd.Flags().Set("tab", "STALE123")
	cmd.Flags().Lookup("tab").Changed = false

	Navigate(client, m.base(), "", "https://pinchtab.com", cmd)

	if len(m.requests) != 2 {
		t.Fatalf("requests = %d, want 2", len(m.requests))
	}
	if m.requests[0].Path != "/tabs" {
		t.Fatalf("first request path = %q, want /tabs", m.requests[0].Path)
	}
	if m.requests[1].Path != "/navigate" {
		t.Fatalf("navigate path = %q, want /navigate", m.requests[1].Path)
	}
}

func TestNavigateWithAllFlags(t *testing.T) {
	m := newMockServer()
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	_ = cmd.Flags().Set("new-tab", "true")
	_ = cmd.Flags().Set("block-images", "true")
	Navigate(client, m.base(), "", "https://pinchtab.com", cmd)
	var body map[string]any
	_ = json.Unmarshal([]byte(m.lastBody), &body)
	if body["newTab"] != true {
		t.Error("expected newTab=true")
	}
	if body["blockImages"] != true {
		t.Error("expected blockImages=true")
	}
}

func TestNavigateWithBlockAds(t *testing.T) {
	m := newMockServer()
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	_ = cmd.Flags().Set("block-ads", "true")
	Navigate(client, m.base(), "", "https://pinchtab.com", cmd)
	var body map[string]any
	_ = json.Unmarshal([]byte(m.lastBody), &body)
	if body["blockAds"] != true {
		t.Error("expected blockAds=true")
	}
}

// TestNavigatePrintTabID verifies that --print-tab-id makes `nav` emit only
// the tab ID on stdout so agents can capture it via `$(pinchtab nav URL)`.
func TestNavigatePrintTabID(t *testing.T) {
	m := newMockServer()
	m.response = `{"tabId":"ABC123","status":"ok"}`
	defer m.close()
	client := m.server.Client()

	cmd := newNavigateCmd()
	_ = cmd.Flags().Set("print-tab-id", "true")

	out := captureStdout(t, func() {
		Navigate(client, m.base(), "", "https://pinchtab.com", cmd)
	})
	got := strings.TrimSpace(out)
	if got != "ABC123" {
		t.Errorf("expected stdout to be exactly 'ABC123', got %q", got)
	}
}
