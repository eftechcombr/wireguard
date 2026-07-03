#!/bin/bash
set -eo pipefail

finish () {
    wg-quick down wg0 || true
    exit 0
}
trap finish SIGTERM SIGINT SIGQUIT

wg-quick up /etc/wireguard/wg0.conf

# Inifinite sleep
sleep infinity &

# healthcheck
python3 wireguard_healthcheck.py & 
wait $!
