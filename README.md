# Wireguard

[![Docker build latest on ghcr](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml/badge.svg)](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml) 

## Overview

This repository provides a lightweight, containerized Wireguard VPN solution built on Alpine Linux. It includes comprehensive health monitoring, automatic key management, and follows Docker best practices for security and reliability.

---

## Features

- ✅ **Lightweight Alpine-based container** (~50MB)
- ✅ **Automated health checks** via HTTP endpoint (port 8080)
- ✅ **Graceful shutdown** with proper signal handling
- ✅ **Easy key generation** using Wireguard tools
- ✅ **Secure by default** with minimal attack surface
- ✅ **Production-ready** with proper logging and monitoring

## Quick Start

### 1. Generate Public and Private Keys

```sh
docker run -i --rm eftechcombr/wireguard wg genkey | tee ./etc/privatekey | docker run -i --rm eftechcombr/wireguard wg pubkey > ./etc/publickey
```

### 2. Configuration

- Place your Wireguard configuration in `etc/wg0.conf`.
- Keys should be stored in `etc/privatekey` and `etc/publickey`.

### 3. Build and Run with Docker

```sh
docker build -t eftechcombr/wireguard .
docker run --rm -it \
  --cap-add=NET_ADMIN \
  --cap-add=SYS_MODULE \
  --sysctl="net.ipv4.conf.all.src_valid_mark=1" \
  -v $(pwd)/etc:/etc/wireguard \
  -v /lib/modules:/lib/modules \
  -p 51820:51820/udp \
  -p 8080:8080/tcp \
  --restart unless-stopped \
  eftechcombr/wireguard
```

### 4. Health Check

The container includes an automated health check server:

```sh
# Check health status
curl http://localhost:8080

# Expected response: HTTP 200 if VPN is active, 503 if down
```

### 5. Peer Connection Logs

The Go service writes structured JSON logs to stdout, including HTTP health-check requests, newly observed peer handshakes, and peer traffic-counter changes. Peer entries include the public key and endpoint; handshake entries also include allowed IPs and the latest handshake time. View logs with:

```sh
docker logs -f wireguard
```

Handshake timestamps are Unix seconds. WireGuard does not expose a definitive peer disconnect event, so these logs report handshakes and traffic changes rather than disconnects. The monitor checks every five seconds by default; set `WG_LOG_INTERVAL` to a Go duration such as `10s` to change the polling interval.

### OpenTelemetry Logs

When an OTLP/HTTP endpoint is configured, the same application logs are exported to OpenTelemetry as well as stdout. Without an endpoint, the service logs to stdout only.

```sh
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318 docker compose up -d
```

You can instead set `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` to configure only the logs endpoint. `OTEL_SERVICE_NAME` defaults to `wireguard`. The exporter uses the standard OTEL headers, timeout, and resource environment settings.

WireGuard itself has no `wg`/`wg-quick` verbosity flag. Linux kernel dynamic-debug messages are a separate host-kernel facility, require kernel support and debugfs access, and go to the kernel log (`dmesg`), not this container's stdout.

### 6. Docker Compose (Recommended)

```yaml
version: '3.8'
services:
  wireguard:
    build: .
    container_name: wireguard
    cap_add:
      - NET_ADMIN
      - SYS_MODULE
    sysctls:
      - net.ipv4.conf.all.src_valid_mark=1
    volumes:
      - ./etc:/etc/wireguard
      - /lib/modules:/lib/modules
    ports:
      - "51820:51820/udp"
      - "8080:8080/tcp"
    restart: unless-stopped
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=UTC
```

---

## Environment Variables

| Variable         | Description                       | Default         |
|------------------|-----------------------------------|-----------------|
| WG_CONF          | Path to Wireguard config file      | /etc/wireguard/wg0.conf |
| WG_PRIVATE_KEY   | Path to private key file           | /etc/wireguard/privatekey |
| WG_PUBLIC_KEY    | Path to public key file            | /etc/wireguard/publickey |
| WG_INTERFACE     | Wireguard interface name           | wg0             |

### Advanced Configuration (Optional)

For production deployments, consider these additional environment variables:

| Variable         | Description                       | Default         |
|------------------|-----------------------------------|-----------------|
| PUID             | User ID for container processes    | 1000            |
| PGID             | Group ID for container processes   | 1000            |
| TZ               | Container timezone                 | UTC             |
| SERVERURL        | External server URL/IP            | auto            |
| SERVERPORT       | External server port              | 51820           |
| INTERNAL_SUBNET  | VPN internal subnet               | 10.13.13.0/24   |
| PEERDNS          | DNS server for clients            | auto            |

---

## File Structure

```
├── etc/
│   ├── privatekey          # Wireguard private key (generated)
│   ├── publickey           # Wireguard public key (generated)
│   └── wg0.conf            # Main Wireguard configuration
├── go.mod                  # Go module definition
├── main.go                 # HTTP health check server (Go)
├── main_test.go            # Health check server unit tests
├── Dockerfile             # Multi-stage container build definition
├── entrypoint.sh          # Container startup script
├── docker-compose.yml     # Orchestration configuration
└── README.md              # This documentation
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Docker Host   │    │   Wireguard     │    │   Health Check  │
│                 │    │   Container     │    │   Server (8080) │
│ ┌─────────────┐ │    │                 │    │                 │
│ │ 51820/UDP   │◄┼────┤ wg0 interface   │    │ HTTP/GET,HEAD  │
│ │ VPN Traffic │ │    │                 │    │ 200/503 status  │
│ └─────────────┘ │    │ ┌─────────────┐ │    │                 │
│ ┌─────────────┐ │    │ │  wg-quick   │ │    │ ┌─────────────┐ │
│ │ 8080/TCP    │◄┼────┤ │    daemon   │ │    │ │  Go         │ │
│ │ Health Check│ │    │ └─────────────┘ │    │ │  HTTP Server│ │
│ └─────────────┘ │    │                 │    │ └─────────────┘ │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

---

## Useful Commands

### Key Management
```sh
# Generate new key pair
docker run -i --rm eftechcombr/wireguard wg genkey | tee ./etc/privatekey | docker run -i --rm eftechcombr/wireguard wg pubkey > ./etc/publickey

# Show current public key
docker run -i --rm eftechcombr/wireguard wg pubkey < ./etc/privatekey
```

### Container Management
```sh
# Build Docker image
docker build -t eftechcombr/wireguard .

# Run with all recommended settings
docker compose up -d

# View container logs
docker logs -f wireguard

# Check VPN status inside container
docker exec wireguard wg show

# Test health endpoint
curl -f http://localhost:8080 || echo "Health check failed"
```

### Monitoring
```sh
# Monitor interface status
watch -n 1 'docker exec wireguard wg show'

# Check system resources
docker stats wireguard
```

---

## Security Considerations

- 🔒 **Container capabilities**: Requires `NET_ADMIN` for network interface management
- 🛡️ **Key protection**: Private keys should be stored securely with proper permissions (600)
- 🔐 **Network isolation**: Consider running in isolated Docker networks
- 📊 **Monitoring**: Use the health check endpoint for load balancer integration
- 🚨 **Signal handling**: Container responds gracefully to SIGTERM/SIGINT

## Troubleshooting

### Common Issues

**Problem**: Container fails to start with permission errors
```sh
# Fix: Ensure proper permissions
chmod 600 ./etc/privatekey
chmod 644 ./etc/publickey
chmod 644 ./etc/wg0.conf
```

**Problem**: VPN traffic not routing
```sh
# Check if interface is up
docker exec wireguard ip link show wg0

# Verify Wireguard status
docker exec wireguard wg show
```

**Problem**: Health check returning 503
```sh
# Verify interface carrier
docker exec wireguard cat /sys/class/net/wg0/carrier

# Expected output: "1" (interface up)
```

## Performance Tuning

- **MTU**: Set appropriate MTU in wg0.conf (typically 1420)
- **Keepalive**: Use `PersistentKeepalive = 25` for NAT traversal
- **DNS**: Configure appropriate DNS servers for client configurations
- **Buffer sizes**: Adjust TCP/UDP buffers for high-throughput scenarios

## References

- [Wireguard Documentation](https://www.wireguard.com/)
- [Wireguard Tools Manual](https://manpages.debian.org/unstable/wireguard-tools/wg.8.en.html)
- [Docker Networking Best Practices](https://docs.docker.com/network/)
- [Linux Security Modules](https://www.kernel.org/doc/html/latest/security/lsm.html)

---

## License

MIT

---

**🐧 Tested on Alpine Linux 3.24 | 🐳 Docker Ready | 🚀 Production Grade**
