# AGENTS.md - Agent Coding Guidelines

## Overview

This is a lightweight, containerized Wireguard VPN solution built on Alpine Linux. The project includes a Go health check server and bash entrypoint script.

## Project Structure

```
.
├── Dockerfile              # Multi-stage Alpine-based container definition
├── docker-compose.yml      # Docker Compose orchestration
├── entrypoint.sh           # Container startup script (bash)
├── go.mod                  # Go module definition
├── main.go                 # HTTP health check server (Go)
├── main_test.go            # Health check server unit tests
├── etc/                    # Wireguard configuration directory
│   ├── wg0.conf           # Main Wireguard configuration
│   ├── privatekey         # Wireguard private key
│   └── publickey          # Wireguard public key
├── .deepsource.toml       # DeepSource linting configuration
└── .github/workflows/     # CI/CD workflows
```

## Build Commands

### Docker Build

```bash
# Build the Docker image
docker build -t eftechcombr/wireguard .

# Build with docker-compose
docker compose build

# Run the container
docker compose up -d
```

### Linting

This project uses **DeepSource** for linting. The configuration is in `.deepsource.toml`.

```bash
# Run DeepSource analyzer (if installed locally)
deepsource analyze

# Or use the DeepSource GitHub integration (configured in repository)
```

### Test Commands

Run Go unit tests:

```bash
# Run unit tests with race detection and coverage
go test -v -race -cover ./...

# Vet Go source code
go vet ./...
```

To run manual integration tests:

```bash
# Test health endpoint after running container
curl http://localhost:8080

# Check Wireguard status inside container
docker exec <container> wg show

# View container logs
docker logs -f <container>
```

## Code Style Guidelines

### General Principles

- Keep files minimal and focused
- Prefer simplicity over complexity
- Use Alpine Linux best practices (minimal packages)

### Go (main.go, main_test.go)

**Formatting:**
- Always format with `gofmt` or `goimports`
- Use standard Go naming conventions (`camelCase`, `PascalCase`)

**Standard Library:**
- Prefer standard library packages (`net/http`, `os`, `os/signal`, `syscall`, `time`)
- Avoid external third-party dependencies unless strictly necessary

**Error Handling:**
- Explicit error checking
- Return appropriate HTTP status codes (200 for success, 503 for failure, 405 for unsupported methods)
- Never expose sensitive errors or stack traces to HTTP clients

**Testing:**
- Write unit tests using standard `testing` package
- Aim for high test coverage of health checking logic and HTTP endpoints

### Shell Scripts (entrypoint.sh)

**Formatting:**
- Use shebang `#!/bin/bash`
- Use 4 spaces for indentation in scripts
- Use lowercase with underscores for variable names

**Best Practices:**
- Always handle signals (SIGTERM, SIGINT) for graceful shutdown
- Use `set -e` for error handling when appropriate
- Quote variables to prevent word splitting
- Use meaningful variable names

**Error Handling:**
- Exit with appropriate codes (0 for success, non-zero for failure)
- Log errors to stderr

**Example:**
```bash
#!/bin/bash

finish() {
    wg-quick down wg0
    exit 0
}
trap finish SIGTERM SIGINT SIGQUIT

wg-quick up /etc/wireguard/wg0.conf
/wireguard_healthcheck &
wait $!
```

### Docker

**Dockerfile Best Practices:**
- Use multi-stage builds to keep final runtime image minimal
- Use specific version tags (e.g., `alpine:3.24`, not `alpine:latest`)
- Combine related RUN commands to reduce layers
- Use `--no-cache` with apk to reduce image size
- Set proper file permissions with `chmod +x`
- Use VOLUME for persistent data
- EXPOSE only necessary ports

**Docker Compose:**
- Use version `3.8` or higher
- Specify all required capabilities (`cap_add`)
- Use restart policies appropriate for production
- Mount volumes with relative paths for portability

### Security

- Never commit private keys or secrets
- Use environment variables for configuration
- Set proper file permissions (600 for keys, 644 for configs)
- Run containers with least privilege principle
- Use specific image tags, not `latest` in production

### Git Workflow

- Create feature branches from `latest`
- Use descriptive commit messages
- Test changes locally before PR
- Ensure Dockerfile builds successfully

### CI/CD

- GitHub Actions workflows are defined in `.github/workflows/`
- Docker images build on push to `dev` and `latest` branches
- Multi-platform builds (linux/amd64, linux/arm64) are configured

## Environment Variables

| Variable    | Description                    | Default                    |
|-------------|--------------------------------|----------------------------|
| WG_CONF     | Path to Wireguard config      | /etc/wireguard/wg0.conf   |
| WG_PRIVATE_KEY | Path to private key        | /etc/wireguard/privatekey |
| WG_PUBLIC_KEY  | Path to public key         | /etc/wireguard/publickey  |
| WG_INTERFACE | Wireguard interface name     | wg0                        |
| PUID        | User ID for processes         | 1000                       |
| PGID        | Group ID for processes        | 1000                       |
| TZ          | Timezone                      | UTC                        |

## Common Tasks

### Adding a New Feature

1. Create a feature branch from `latest`
2. Make changes following code style guidelines
3. Test locally with `docker compose up`
4. Verify health endpoint responds correctly
5. Submit pull request

### Debugging

```bash
# Check container logs
docker logs wireguard

# Exec into container
docker exec -it wireguard /bin/sh

# Check Wireguard interface
docker exec wireguard wg show

# Test health endpoint
curl -v http://localhost:8080
```

### Key Management

```bash
# Generate new key pair
docker run -i --rm eftechcombr/wireguard wg genkey | tee ./etc/privatekey | \
  docker run -i --rm eftechcombr/wireguard wg pubkey > ./etc/publickey
```
