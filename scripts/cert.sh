#!/bin/bash

openssl req -newkey rsa:2048 -nodes -keyout tls.key -x509 -days 1095 -subj '/CN=localhost' -out tls.crt
