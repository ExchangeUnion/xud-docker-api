#!/bin/bash

set -euo pipefail

SCRIPTS=$(dirname "$0")
NETWORK=${1:-testnet}


if [[ $(uname) == "Darwin" ]]; then
  NETWORK_DIR="$HOME/Library/Application Support/XudDocker/$NETWORK"
else
  NETWORK_DIR="$HOME/.xud-docker/$NETWORK"
fi
DATA_DIR="$NETWORK_DIR/data"
PROXY_DIR="$DATA_DIR/proxy"

echo "PROXY_DIR=$PROXY_DIR"

SERVICES_JSON="$(base64 -i "$SCRIPTS/${NETWORK}.json")"

docker build . -t proxy
docker run -it --rm --name proxy \
-e "NETWORK=$NETWORK" \
-e "SERVICES=$SERVICES_JSON" \
--net "${NETWORK}_default" \
-p 8080:8080 \
-v /var/run/docker.sock:/var/run/docker.sock \
-v "$PROXY_DIR:/root/.proxy" \
-v "$NETWORK_DIR:/root/network:ro" \
-v "$HOME/xud-ui-dashboard/build:/ui" \
proxy
