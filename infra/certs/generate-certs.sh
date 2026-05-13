#!/bin/bash
# generate-certs.sh - Generate self-signed CA and service certificates for development.
# Usage: ./generate-certs.sh
# Output: infra/certs/dev/

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_DIR="${SCRIPT_DIR}/dev"
DAYS=365

SERVICES=(gateway reservation billing payment presence notification search)

echo "==> Generating development mTLS certificates"
echo "    Output: ${OUT_DIR}"

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

# --- Generate CA ---
echo "==> Generating CA key and certificate"
openssl ecparam -genkey -name prime256v1 -noout -out "${OUT_DIR}/ca-key.pem" 2>/dev/null
openssl req -new -x509 -key "${OUT_DIR}/ca-key.pem" \
  -out "${OUT_DIR}/ca.pem" \
  -days "${DAYS}" \
  -subj "/O=ParkirPintar/CN=ParkirPintar Dev CA" \
  2>/dev/null

echo "    CA certificate: ${OUT_DIR}/ca.pem"

# --- Generate per-service server certificates ---
for svc in "${SERVICES[@]}"; do
  echo "==> Generating certificate for service: ${svc}"

  # Generate key
  openssl ecparam -genkey -name prime256v1 -noout -out "${OUT_DIR}/${svc}-key.pem" 2>/dev/null

  # Create CSR config with SANs
  cat > "${OUT_DIR}/${svc}-csr.conf" <<EOF
[req]
default_bits = 256
prompt = no
distinguished_name = dn
req_extensions = v3_req

[dn]
O = ParkirPintar
CN = ${svc}

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${svc}
DNS.2 = ${svc}.parkir-pintar
DNS.3 = ${svc}.parkir-pintar.svc.cluster.local
DNS.4 = localhost
IP.1 = 127.0.0.1
EOF

  # Generate CSR
  openssl req -new -key "${OUT_DIR}/${svc}-key.pem" \
    -out "${OUT_DIR}/${svc}.csr" \
    -config "${OUT_DIR}/${svc}-csr.conf" \
    2>/dev/null

  # Sign with CA
  openssl x509 -req -in "${OUT_DIR}/${svc}.csr" \
    -CA "${OUT_DIR}/ca.pem" \
    -CAkey "${OUT_DIR}/ca-key.pem" \
    -CAcreateserial \
    -out "${OUT_DIR}/${svc}-cert.pem" \
    -days "${DAYS}" \
    -extensions v3_req \
    -extfile "${OUT_DIR}/${svc}-csr.conf" \
    2>/dev/null

  # Cleanup CSR and config
  rm -f "${OUT_DIR}/${svc}.csr" "${OUT_DIR}/${svc}-csr.conf"

  echo "    Cert: ${OUT_DIR}/${svc}-cert.pem"
  echo "    Key:  ${OUT_DIR}/${svc}-key.pem"
done

# --- Generate client certificate for inter-service communication ---
echo "==> Generating client certificate for inter-service communication"

openssl ecparam -genkey -name prime256v1 -noout -out "${OUT_DIR}/client-key.pem" 2>/dev/null

cat > "${OUT_DIR}/client-csr.conf" <<EOF
[req]
default_bits = 256
prompt = no
distinguished_name = dn
req_extensions = v3_req

[dn]
O = ParkirPintar
CN = parkir-pintar-client

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

openssl req -new -key "${OUT_DIR}/client-key.pem" \
  -out "${OUT_DIR}/client.csr" \
  -config "${OUT_DIR}/client-csr.conf" \
  2>/dev/null

openssl x509 -req -in "${OUT_DIR}/client.csr" \
  -CA "${OUT_DIR}/ca.pem" \
  -CAkey "${OUT_DIR}/ca-key.pem" \
  -CAcreateserial \
  -out "${OUT_DIR}/client-cert.pem" \
  -days "${DAYS}" \
  -extensions v3_req \
  -extfile "${OUT_DIR}/client-csr.conf" \
  2>/dev/null

rm -f "${OUT_DIR}/client.csr" "${OUT_DIR}/client-csr.conf" "${OUT_DIR}/ca.srl"

echo "    Client cert: ${OUT_DIR}/client-cert.pem"
echo "    Client key:  ${OUT_DIR}/client-key.pem"

echo ""
echo "==> Done! All certificates generated in ${OUT_DIR}"
echo "    CA:     ca.pem / ca-key.pem"
echo "    Server: <service>-cert.pem / <service>-key.pem"
echo "    Client: client-cert.pem / client-key.pem"
