package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	defaultInterface = "wg0"
	defaultPort      = "8080"
	serverHeader     = "meow! You shall not pass!"
	htmlContent      = `
        <html><head><title>Wireguard Health Check</title></head>
        <body>
        <pre>
            meow!
        </pre>
        </body></html>
        `
)

// HealthChecker checks the status of the WireGuard interface and handles HTTP requests.
type HealthChecker struct {
	InterfaceName string
	SysClassNet   string
}

// IsLinkUp checks if the network interface carrier indicates the link is up.
func (h *HealthChecker) IsLinkUp() bool {
	basePath := h.SysClassNet
	if basePath == "" {
		basePath = "/sys/class/net"
	}
	carrierPath := filepath.Join(basePath, h.InterfaceName, "carrier")
	data, err := os.ReadFile(carrierPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) == "1"
}

// ServeHTTP implements http.Handler for WireGuard health check.
func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Server", serverHeader)

	statusCode := http.StatusOK
	if !h.IsLinkUp() {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)

	if r.Method == http.MethodGet {
		_, _ = w.Write([]byte(htmlContent))
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func main() {
	iface := getEnvOrDefault("WG_INTERFACE", defaultInterface)
	port := getEnvOrDefault("HEALTHCHECK_PORT", getEnvOrDefault("PORT", defaultPort))
	sysClassNet := os.Getenv("SYS_CLASS_NET")

	checker := &HealthChecker{
		InterfaceName: iface,
		SysClassNet:   sysClassNet,
	}

	server := newServer(fmt.Sprintf(":%s", port), checker)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		log.Printf("Starting WireGuard health check server on :%s (interface: %s)", port, iface)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down health check server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
