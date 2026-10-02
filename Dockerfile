FROM golang:alpine3.24 AS builder

WORKDIR /src

COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /wireguard_healthcheck .

FROM alpine:3.24

COPY --from=builder /wireguard_healthcheck /wireguard_healthcheck
COPY entrypoint.sh /

RUN apk add --no-cache \
    wireguard-tools=~1.0.20260223 \
    curl \
    iptables && \
  chmod +x /entrypoint.sh /wireguard_healthcheck

VOLUME [ "/etc/wireguard" ]

EXPOSE 51820/UDP 8080/TCP

ENTRYPOINT [ "/entrypoint.sh" ]
