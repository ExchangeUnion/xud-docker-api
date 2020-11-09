#!/bin/bash

openssl req -newkey rsa:2048 -nodes -keyout xudproxy.key -x509 -days 1095 -subj '/CN=localhost' -out xudproxy.crt
