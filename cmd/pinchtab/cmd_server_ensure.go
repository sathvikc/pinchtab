package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"time"
)

const ensureServerTimeout = 30 * time.Second

type serverStartFunc func() error
type serverHealthFunc func(baseURL, token string) bool

func ensureServer(baseURL, token, command string) error {
	return ensureServerWith(baseURL, token, command, autoStartServer, isServerHealthy, ensureServerTimeout)
}

func ensureServerWith(baseURL, token, command string, start serverStartFunc, healthy serverHealthFunc, timeout time.Duration) error {
	if healthy(baseURL, token) {
		return nil
	}

	slog.Info("server not running, starting automatically", "url", baseURL, "command", command)
	if err := start(); err != nil {
		slog.Error("failed to auto-start server", "err", err, "command", command)
		return fmt.Errorf("server at %s is not running and auto-start failed: %w", baseURL, err)
	}

	if !waitForServerWith(baseURL, token, timeout, healthy) {
		return fmt.Errorf("server did not become healthy at %s within %s", baseURL, timeout)
	}

	slog.Info("server started successfully", "url", baseURL, "command", command)
	return nil
}

func isServerHealthy(baseURL, token string) bool {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return false
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode < 500
}

func autoStartServer() error {
	binary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	args := []string{"server"}
	if serverURL != "" {
		args = []string{"--server", serverURL, "server"}
	}

	cmd := exec.Command(binary, args...) // #nosec G204 -- binary is our own executable from os.Executable(), args are hardcoded subcommands
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	detachProcess(cmd)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn server: %w", err)
	}

	if err := cmd.Process.Release(); err != nil {
		slog.Warn("failed to release server process", "err", err)
	}

	return nil
}

func waitForServer(baseURL, token string, timeout time.Duration) bool {
	return waitForServerWith(baseURL, token, timeout, isServerHealthy)
}

func waitForServerWith(baseURL, token string, timeout time.Duration, healthy serverHealthFunc) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if healthy(baseURL, token) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
