FROM alpine:3.20

COPY wireguard_healthcheck.py /
COPY entrypoint.sh / 

RUN apk update && \
  apk --no-cache add wireguard-tools=~1.0.20210914 && \
  apk --no-cache add python3=~3.12 && \
  apk --no-cache add curl=~8.9 && \
  apk --no-cache add iptables=~1.8 && \
  chmod +x entrypoint.sh wireguard_healthcheck.py

VOLUME [ "/etc/wireguard" ]

ENTRYPOINT [ "/entrypoint.sh" ]

EXPOSE 51820/UDP 8080/TCP

