# Wireguard

[![Docker build latest on ghcr](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml/badge.svg)](https://github.com/eftechcombr/wireguard/actions/workflows/docker-publish-ghcr-latest.yml) 


> ## Generate Public and Private Key
    
    docker run -i --rm eftechcombr/wireguard wg genkey | tee ./privatekey | docker run -i --rm eftechcombr/wireguard wg pubkey > ./publickey


