FROM alpine:3.22

COPY wireguard_healthcheck.py /
COPY entrypoint.sh / 

RUN apk add --no-cache \
    wireguard-tools=~1.0.20250521 \
    python3 \
    curl \
    iptables && \
  chmod +x /entrypoint.sh /wireguard_healthcheck.py

VOLUME [ "/etc/wireguard" ]

EXPOSE 51820/UDP 8080/TCP

ENTRYPOINT [ "/entrypoint.sh" ]


