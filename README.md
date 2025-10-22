# Wireguard

[![Docker build latest on ghcr](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml/badge.svg)](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml) 

## Overview

This repository provides a containerized Wireguard VPN solution, including health checks.

---

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
  -v $(pwd)/etc:/etc/wireguard \
  eftechcombr/wireguard
```

### 4. Health Check

A health check script is provided:

```sh
python wireguard_healthcheck.py
```

---

## Environment Variables

| Variable         | Description                       | Default         |
|------------------|-----------------------------------|-----------------|
| WG_CONF          | Path to Wireguard config file      | /etc/wireguard/wg0.conf |
| WG_PRIVATE_KEY   | Path to private key file           | /etc/wireguard/privatekey |
| WG_PUBLIC_KEY    | Path to public key file            | /etc/wireguard/publickey |
| WG_INTERFACE     | Wireguard interface name           | wg0             |

---

## File Structure

```
etc/
  privatekey
  publickey
  wg0.conf
wireguard_healthcheck.py
Dockerfile
entrypoint.sh
```

---

## Useful Commands

- Generate keys: see above
- Run health check: `python wireguard_healthcheck.py`
- Build Docker image: `docker build -t eftechcombr/wireguard .`
- Run container: see above

---

## References

- [Wireguard Documentation](https://www.wireguard.com/)

---

## License

MIT


