#!/bin/bash

docker build . -t proxy
docker run -it --rm --name proxy \
--net testnet_default \
-p 8080:8080 \
-v /var/run/docker.sock:/var/run/docker.sock \
-v $HOME/.xud-docker/testnet/data/xud:/root/.xud \
proxy