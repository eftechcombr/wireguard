package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsLinkUp(t *testing.T) {
	tempDir := t.TempDir()
	iface := "wg0"
	ifaceDir := filepath.Join(tempDir, iface)
	if err := os.MkdirAll(ifaceDir, 0750); err != nil {
		t.Fatalf("failed to create temp iface dir: %v", err)
	}

	checker := &HealthChecker{
		InterfaceName: iface,
		SysClassNet:   tempDir,
	}

	// Case 1: Missing carrier file -> false
	if checker.IsLinkUp() {
		t.Errorf("expected IsLinkUp to be false when carrier file is missing")
	}

	// Case 2: Carrier file contains '0' -> false
	carrierFile := filepath.Join(ifaceDir, "carrier")
	if err := os.WriteFile(carrierFile, []byte("0\n"), 0644); err != nil {
		t.Fatalf("failed to write carrier file: %v", err)
	}
	if checker.IsLinkUp() {
		t.Errorf("expected IsLinkUp to be false when carrier is '0'")
	}

	// Case 3: Carrier file contains '1' with whitespace -> true
	if err := os.WriteFile(carrierFile, []byte("1\n"), 0644); err != nil {
		t.Fatalf("failed to write carrier file: %v", err)
	}
	if !checker.IsLinkUp() {
		t.Errorf("expected IsLinkUp to be true when carrier is '1'")
	}

	// Case 4: Carrier file contains random text -> false
	if err := os.WriteFile(carrierFile, []byte("error"), 0644); err != nil {
		t.Fatalf("failed to write carrier file: %v", err)
	}
	if checker.IsLinkUp() {
		t.Errorf("expected IsLinkUp to be false when carrier is 'error'")
	}
}

func TestServeHTTP_GET(t *testing.T) {
	tempDir := t.TempDir()
	iface := "wg0"
	ifaceDir := filepath.Join(tempDir, iface)
	if err := os.MkdirAll(ifaceDir, 0750); err != nil {
		t.Fatalf("failed to create temp iface dir: %v", err)
	}
	carrierFile := filepath.Join(ifaceDir, "carrier")

	checker := &HealthChecker{
		InterfaceName: iface,
		SysClassNet:   tempDir,
	}

	// Test GET when link is down (503)
	if err := os.WriteFile(carrierFile, []byte("0\n"), 0644); err != nil {
		t.Fatalf("failed to write carrier: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	checker.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
	if s := res.Header.Get("Server"); s != serverHeader {
		t.Errorf("expected Server header %q, got %q", serverHeader, s)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if !strings.Contains(string(body), "meow!") {
		t.Errorf("expected body to contain 'meow!', got %q", string(body))
	}

	// Test GET when link is up (200)
	if err := os.WriteFile(carrierFile, []byte("1\n"), 0644); err != nil {
		t.Fatalf("failed to write carrier: %v", err)
	}

	rr2 := httptest.NewRecorder()
	checker.ServeHTTP(rr2, req)

	res2 := rr2.Result()
	defer res2.Body.Close()

	if res2.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res2.StatusCode)
	}
	if ct := res2.Header.Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
	if s := res2.Header.Get("Server"); s != serverHeader {
		t.Errorf("expected Server header %q, got %q", serverHeader, s)
	}

	body2, _ := io.ReadAll(res2.Body)
	if !strings.Contains(string(body2), "meow!") {
		t.Errorf("expected body to contain 'meow!', got %q", string(body2))
	}
}

func TestServeHTTP_HEAD(t *testing.T) {
	tempDir := t.TempDir()
	iface := "wg0"
	ifaceDir := filepath.Join(tempDir, iface)
	if err := os.MkdirAll(ifaceDir, 0750); err != nil {
		t.Fatalf("failed to create temp iface dir: %v", err)
	}
	carrierFile := filepath.Join(ifaceDir, "carrier")

	checker := &HealthChecker{
		InterfaceName: iface,
		SysClassNet:   tempDir,
	}

	// Link is UP
	if err := os.WriteFile(carrierFile, []byte("1\n"), 0644); err != nil {
		t.Fatalf("failed to write carrier: %v", err)
	}

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rr := httptest.NewRecorder()
	checker.ServeHTTP(rr, req)

	res := rr.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
	if s := res.Header.Get("Server"); s != serverHeader {
		t.Errorf("expected Server header %q, got %q", serverHeader, s)
	}

	body, _ := io.ReadAll(res.Body)
	if len(body) != 0 {
		t.Errorf("expected empty body for HEAD request, got %d bytes: %q", len(body), string(body))
	}
}

func TestServeHTTP_UnsupportedMethods(t *testing.T) {
	checker := &HealthChecker{
		InterfaceName: "wg0",
	}

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rr := httptest.NewRecorder()
			checker.ServeHTTP(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("expected 405 for %s, got %d", method, res.StatusCode)
			}
			if allow := res.Header.Get("Allow"); allow != "GET, HEAD" {
				t.Errorf("expected Allow header 'GET, HEAD', got %q", allow)
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	key := "TEST_ENV_VAR_WIREGUARD"

	if val := getEnvOrDefault(key, "fallback"); val != "fallback" {
		t.Errorf("expected fallback, got %q", val)
	}

	t.Setenv(key, "custom")

	if val := getEnvOrDefault(key, "fallback"); val != "custom" {
		t.Errorf("expected custom, got %q", val)
	}
}

func TestNewServer(t *testing.T) {
	checker := &HealthChecker{InterfaceName: "wg0"}
	srv := newServer(":8080", checker)
	if srv.Addr != ":8080" {
		t.Errorf("expected :8080, got %s", srv.Addr)
	}
	if srv.Handler != checker {
		t.Errorf("expected handler to be checker")
	}
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("expected 5s ReadHeaderTimeout, got %v", srv.ReadHeaderTimeout)
	}
}

