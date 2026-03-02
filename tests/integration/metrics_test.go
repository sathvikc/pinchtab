//go:build integration

package integration

import (
	"encoding/json"
	"testing"
)

// M1: Get metrics endpoint returns aggregated memory across instances
func TestMetrics_Basic(t *testing.T) {
	// Navigate to a page first to ensure we have a tab with content
	navigate(t, "https://example.com")
	defer closeCurrentTab(t)

	code, body := httpGet(t, "/instances/metrics")
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, string(body))
	}

	var metrics []map[string]any
	if err := json.Unmarshal(body, &metrics); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	// Should have at least one instance with metrics
	if len(metrics) == 0 {
		t.Fatal("expected at least one instance in metrics response")
	}

	// Check first instance has expected fields
	m := metrics[0]
	fields := []string{"instanceId", "profileName", "jsHeapUsedMB", "jsHeapTotalMB"}
	for _, f := range fields {
		if _, ok := m[f]; !ok {
			t.Errorf("expected %s in metrics response", f)
		}
	}
}

// M2: Per-tab metrics (proxied through orchestrator)
func TestMetrics_PerTab(t *testing.T) {
	navigate(t, "https://example.com")
	defer closeCurrentTab(t)

	code, body := httpGet(t, "/tabs/"+currentTabID+"/metrics")
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, string(body))
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	// Should have memoryMB (OS-level) or jsHeapUsedMB (estimated)
	_, hasMemory := m["memoryMB"].(float64)
	_, hasHeap := m["jsHeapUsedMB"].(float64)
	if !hasMemory && !hasHeap {
		t.Fatal("expected memoryMB or jsHeapUsedMB in response")
	}
}

// M3: Invalid tab ID returns error
func TestMetrics_InvalidTab(t *testing.T) {
	code, _ := httpGet(t, "/tabs/invalid_tab_id/metrics")
	if code != 500 && code != 404 {
		t.Errorf("expected 500 or 404 for invalid tab, got %d", code)
	}
}

// M4: Metrics endpoint returns valid data
func TestMetrics_ValidResponse(t *testing.T) {
	navigate(t, "https://example.com")
	defer closeCurrentTab(t)

	code, body := httpGet(t, "/tabs/"+currentTabID+"/metrics")
	if code != 200 {
		t.Fatalf("expected 200, got %d: %s", code, string(body))
	}

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	// Verify response has expected structure (either OS-level or estimated metrics)
	if _, ok := m["memoryMB"]; !ok {
		if _, ok := m["jsHeapUsedMB"]; !ok {
			t.Error("expected memoryMB or jsHeapUsedMB in response")
		}
	}
}
