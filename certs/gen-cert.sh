#!/bin/bash
set -e

mkdir -p certs

echo "creating CA private key..."
openssl genrsa -out ./certs/ca.key 2048

echo "creating self-signed certificate authority based on the private key CA..."
openssl req -new -x509 -key ./certs/ca.key -days 365 -out ./certs/ca.crt -subj "/C=US/ST=State/L=City/O=Example/OU=IT/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:0.0.0.0"

echo "creating server private key..."
openssl genrsa -out ./certs/server.key 2048

echo "creating server certificate signing request (CSR)..."
openssl req -new -key ./certs/server.key -out ./certs/server.csr -subj "/C=US/ST=State/L=City/O=Example/OU=IT/CN=localhost" -batch -addext "subjectAltName=DNS:localhost,IP:0.0.0.0"

echo "signed server csr by CA..."
openssl x509 -req -in ./certs/server.csr -CA ./certs/ca.crt -CAkey ./certs/ca.key -CAcreateserial -out ./certs/server.crt -days 365 -sha256 -extfile <(echo "subjectAltName=DNS:localhost,IP:0.0.0.0")
