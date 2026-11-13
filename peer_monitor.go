package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type peerStats struct {
	endpoint        string
	allowedIPs      string
	latestHandshake int64
	receivedBytes   uint64
	sentBytes       uint64
}

func getLogInterval() time.Duration {
	value := os.Getenv("WG_LOG_INTERVAL")
	if value == "" {
		return 5 * time.Second
	}
	interval, err := time.ParseDuration(value)
	if err != nil || interval <= 0 {
		return 5 * time.Second
	}
	return interval
}

func monitorWireGuardPeers(ctx context.Context, iface string, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var previous map[string]peerStats
	for {
		current, err := readPeerStats(ctx, iface)
		if err != nil {
			if ctx.Err() == nil {
				logger.WarnContext(ctx, "Unable to read WireGuard peer status",
					slog.String("interface", iface),
					slog.String("error", err.Error()),
				)
			}
		} else if previous != nil {
			logPeerChanges(ctx, logger, previous, current)
		}
		if err == nil {
			previous = current
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func readPeerStats(ctx context.Context, iface string) (map[string]peerStats, error) {
	handshakes, err := readWGValues(ctx, iface, "latest-handshakes")
	if err != nil {
		return nil, err
	}
	transfers, err := readWGValues(ctx, iface, "transfer")
	if err != nil {
		return nil, err
	}
	endpoints, err := readWGValues(ctx, iface, "endpoints")
	if err != nil {
		return nil, err
	}
	allowedIPs, err := readWGValues(ctx, iface, "allowed-ips")
	if err != nil {
		return nil, err
	}

	peers := make(map[string]peerStats, len(handshakes))
	for publicKey, value := range handshakes {
		handshake, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse handshake timestamp: %w", err)
		}
		transferFields := strings.Fields(transfers[publicKey])
		if len(transferFields) != 2 {
			return nil, fmt.Errorf("missing transfer counters for peer %s", publicKey)
		}
		received, err := strconv.ParseUint(transferFields[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse received byte count: %w", err)
		}
		sent, err := strconv.ParseUint(transferFields[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse sent byte count: %w", err)
		}
		peers[publicKey] = peerStats{
			endpoint:        endpoints[publicKey],
			allowedIPs:      allowedIPs[publicKey],
			latestHandshake: handshake,
			receivedBytes:   received,
			sentBytes:       sent,
		}
	}
	return peers, nil
}

func readWGValues(ctx context.Context, iface, field string) (map[string]string, error) {
	output, err := exec.CommandContext(ctx, "wg", "show", iface, field).Output()
	if err != nil {
		return nil, fmt.Errorf("wg show %s %s: %w", iface, field, err)
	}
	values := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			values[fields[0]] = strings.Join(fields[1:], " ")
		}
	}
	return values, nil
}

func logPeerChanges(ctx context.Context, logger *slog.Logger, previous, current map[string]peerStats) {
	for publicKey, peer := range current {
		last, exists := previous[publicKey]
		if !exists {
			continue
		}
		if peer.latestHandshake > last.latestHandshake {
			logger.InfoContext(ctx, "WireGuard peer handshake",
				slog.String("peer.public_key", publicKey),
				slog.String("peer.endpoint", peer.endpoint),
				slog.String("peer.allowed_ips", peer.allowedIPs),
				slog.Int64("peer.latest_handshake", peer.latestHandshake),
			)
		}
		if peer.receivedBytes != last.receivedBytes || peer.sentBytes != last.sentBytes {
			logger.InfoContext(ctx, "WireGuard peer traffic",
				slog.String("peer.public_key", publicKey),
				slog.String("peer.endpoint", peer.endpoint),
				slog.Int64("peer.received_bytes_delta", int64(peer.receivedBytes)-int64(last.receivedBytes)),
				slog.Uint64("peer.received_bytes_total", peer.receivedBytes),
				slog.Int64("peer.sent_bytes_delta", int64(peer.sentBytes)-int64(last.sentBytes)),
				slog.Uint64("peer.sent_bytes_total", peer.sentBytes),
			)
		}
	}
}
