//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

var (
	serverURL    string
	currentTabID string // Track current tab for action operations
)

func TestMain(m *testing.M) {
	port := os.Getenv("PINCHTAB_TEST_PORT")
	if port == "" {
		port = "19867"
	}
	serverURL = fmt.Sprintf("http://localhost:%s", port)

	// Single parent temp dir for all test artifacts (binary, state, profiles).
	// Cleaned up explicitly before os.Exit — defer won't run after os.Exit.
	testDir, err := os.MkdirTemp("", "pinchtab-test-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create test dir: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "TestMain: test dir: %s\n", testDir)

	cleanup := func() {
		if os.Getenv("PINCHTAB_TEST_KEEP_DIR") != "" {
			fmt.Fprintf(os.Stderr, "TestMain: keeping test dir (PINCHTAB_TEST_KEEP_DIR set): %s\n", testDir)
			return
		}
		os.RemoveAll(testDir)
	}

	binaryPath := filepath.Join(testDir, "pinchtab")
	stateDir := filepath.Join(testDir, "state")
	profileDir := filepath.Join(testDir, "profiles")

	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create state dir: %v\n", err)
		cleanup()
		os.Exit(1)
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create profile dir: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	// Build the binary into the test dir
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/pinchtab/")
	build.Dir = findRepoRoot()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build pinchtab: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	// Start server in its own process group so we can kill Chrome children on shutdown.
	cmd := exec.Command(binaryPath)

	// Build environment for subprocess
	// Start with a filtered set of inherited env vars, then add test-specific ones
	baseEnv := os.Environ()
	env := []string{}
	for _, e := range baseEnv {
		// Skip any pre-existing BRIDGE_* and PINCHTAB_* vars to avoid conflicts
		if !strings.HasPrefix(e, "BRIDGE_") && !strings.HasPrefix(e, "PINCHTAB_") {
			env = append(env, e)
		}
	}

	// Add test-specific environment (use PINCHTAB_* names, BRIDGE_* are deprecated)
	env = append(env,
		"PINCHTAB_PORT="+port,
		"PINCHTAB_HEADLESS=true",
		"PINCHTAB_NO_RESTORE=true",
		"PINCHTAB_STEALTH=light",
		"PINCHTAB_STATE_DIR="+stateDir,
		"PINCHTAB_PROFILE_DIR="+profileDir,
	)

	// Pass CHROME_BINARY if set by CI workflow or environment
	if chromeBinary := os.Getenv("CHROME_BINARY"); chromeBinary != "" {
		env = append(env, "CHROME_BINARY="+chromeBinary)
	}

	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start pinchtab: %v\n", err)
		cleanup()
		os.Exit(1)
	}

	// Wait for server to be ready (longer timeout in CI)
	healthTimeout := 30 * time.Second
	if os.Getenv("CI") == "true" {
		healthTimeout = 60 * time.Second
	}
	if !waitForHealth(serverURL, healthTimeout) {
		fmt.Fprintf(os.Stderr, "pinchtab did not become healthy within timeout (%v)\n", healthTimeout)
		_ = cmd.Process.Kill()
		cleanup()
		os.Exit(1)
	}

	// Launch a test instance for orchestrator-mode tests
	// This ensures /navigate and other proxy endpoints work in CI
	if err := launchTestInstance(serverURL); err != nil {
		fmt.Fprintf(os.Stderr, "failed to launch test instance: %v\n", err)
		_ = cmd.Process.Kill()
		cleanup()
		os.Exit(1)
	}

	code := m.Run()

	// Shutdown — send SIGTERM to the process group to also kill Chrome children.
	if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	} else {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		if pgid, err := syscall.Getpgid(cmd.Process.Pid); err == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
		_ = cmd.Process.Kill()
	}

	cleanup()
	os.Exit(code)
}

func findRepoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Join("..", "..")
}

func waitForHealth(base string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil && resp.StatusCode == 200 {
			_ = resp.Body.Close()
			return true
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// helpers

func httpGet(t *testing.T, path string) (int, []byte) {
	t.Helper()
	resp, err := http.Get(serverURL + path)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func httpPost(t *testing.T, path string, payload any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		reader = strings.NewReader(string(data))
	}
	resp, err := http.Post(serverURL+path, "application/json", reader)
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func httpPostRaw(t *testing.T, path string, body string) (int, []byte) {
	t.Helper()
	resp, err := http.Post(serverURL+path, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s failed: %v", path, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}

func jsonField(t *testing.T, data []byte, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json parse failed: %v (body: %s)", err, string(data))
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		b, _ := json.Marshal(val)
		return string(b)
	}
}

func navigate(t *testing.T, url string) {
	t.Helper()
	// Use retry logic for better stability
	code, body := httpPostWithRetry(t, "/navigate", map[string]any{"url": url}, 2)
	if code != 200 {
		t.Fatalf("navigate to %s failed with %d: %s", url, code, string(body))
	}

	// Extract tabId from response for subsequent action calls
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		t.Logf("warning: failed to parse navigate response: %v", err)
		return
	}

	if id, ok := result["tabId"].(string); ok {
		currentTabID = id
		t.Logf("current tab: %s", currentTabID)
		// Auto-close tab on test completion to prevent Chrome hitting tab limit.
		// Safe to call even if the test also defers closeCurrentTab (idempotent).
		t.Cleanup(func() { closeCurrentTab(t) })
	}
}

// closeCurrentTab closes the current tab to clean up resources
func closeCurrentTab(t *testing.T) {
	t.Helper()
	if currentTabID == "" {
		return
	}
	// Close the tab
	_, _ = httpPost(t, "/tab", map[string]any{
		"tabId":  currentTabID,
		"action": "close",
	})
	currentTabID = ""
}

// navigateInstance creates a fresh tab on the given instance and navigates it.
func navigateInstance(t *testing.T, instID, url string) (int, []byte, string) {
	t.Helper()

	openCode, openBody := httpPostWithRetry(t, fmt.Sprintf("/instances/%s/tabs/open", instID), map[string]any{
		"url": "about:blank",
	}, 2)
	if openCode != 200 {
		return openCode, openBody, ""
	}

	tabID := jsonField(t, openBody, "tabId")
	if tabID == "" {
		return 500, []byte(`{"error":"missing tabId from open tab response"}`), ""
	}

	path := fmt.Sprintf("/tabs/%s/navigate", tabID)
	code, body := httpPostWithRetry(t, path, map[string]any{"url": url}, 2)
	return code, body, tabID
}

// waitForInstanceReady waits for an instance to be ready for navigation
// Uses a simple navigate to about:blank to check readiness
func waitForInstanceReady(t *testing.T, instID string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		code, _, _ := navigateInstance(t, instID, "about:blank")
		if code == 200 {
			t.Logf("instance %s is ready", instID)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Logf("warning: instance %s did not become ready within 15 seconds", instID)
}

// launchTestInstance launches a default test instance for orchestrator-mode tests
// This is called once during TestMain setup so that /navigate and proxy endpoints work
// It waits for the instance to be fully ready before returning
func launchTestInstance(base string) error {
	resp, err := http.Post(
		base+"/instances/launch",
		"application/json",
		strings.NewReader(`{"mode":"headless"}`),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("launch failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get instance ID
	var result map[string]any
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse launch response: %v", err)
	}

	id, ok := result["id"].(string)
	if !ok {
		return fmt.Errorf("no instance id in launch response: %v", result)
	}

	fmt.Fprintf(os.Stderr, "TestMain: launched test instance %s\n", id)

	// Wait for instance to be ready (can take 2-5 seconds for Chrome to start)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		// Open a tab first, then navigate it to check if instance is ready.
		openResp, err := http.Post(
			base+"/instances/"+id+"/tabs/open",
			"application/json",
			strings.NewReader(`{"url":"about:blank"}`),
		)
		if err == nil {
			var tabID string
			if openResp.StatusCode == 200 {
				openBody, _ := io.ReadAll(openResp.Body)
				var open map[string]any
				if err := json.Unmarshal(openBody, &open); err == nil {
					if idValue, ok := open["tabId"].(string); ok {
						tabID = idValue
					}
				}
			}
			_ = openResp.Body.Close()
			if tabID != "" {
				navResp, err := http.Post(
					base+"/tabs/"+tabID+"/navigate",
					"application/json",
					strings.NewReader(`{"url":"about:blank"}`),
				)
				if err == nil {
					if navResp.StatusCode == 200 {
						_ = navResp.Body.Close()
						fmt.Fprintf(os.Stderr, "TestMain: instance %s is ready\n", id)
						return nil
					}
					navBody, _ := io.ReadAll(navResp.Body)
					_ = navResp.Body.Close()
					fmt.Fprintf(os.Stderr, "TestMain: instance not ready yet (%d): %s\n", navResp.StatusCode, string(navBody))
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("instance %s did not become ready within 30 seconds", id)
}
