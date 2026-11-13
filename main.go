// Package main provides a lightweight HTTP health check server for WireGuard.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		logger.InfoContext(r.Context(), "HTTP request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

func main() {
	iface := getEnvOrDefault("WG_INTERFACE", defaultInterface)
	port := getEnvOrDefault("HEALTHCHECK_PORT", getEnvOrDefault("PORT", defaultPort))
	sysClassNet := os.Getenv("SYS_CLASS_NET")

	logger, shutdownLogger, err := newLogger(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize logger: %v\n", err)
		os.Exit(1)
	}

	checker := &HealthChecker{
		InterfaceName: iface,
		SysClassNet:   sysClassNet,
	}

	server := newServer(fmt.Sprintf(":%s", port), requestLogger(logger, checker))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(stop)

	monitorCtx, cancelMonitor := context.WithCancel(context.Background())
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		monitorWireGuardPeers(monitorCtx, iface, getLogInterval(), logger)
	}()
	go func() {
		logger.Info("Starting WireGuard health check server",
			slog.String("address", fmt.Sprintf(":%s", port)),
			slog.String("interface", iface),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Health check server failed", slog.String("error", err.Error()))
			select {
			case stop <- syscall.SIGTERM:
			default:
			}
		}
	}()

	<-stop
	logger.Info("Shutting down health check server")
	cancelMonitor()
	<-monitorDone

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Health check server shutdown failed", slog.String("error", err.Error()))
	}
	if err := shutdownLogger(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown logger: %v\n", err)
	}
}
