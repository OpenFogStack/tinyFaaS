#!/bin/bash

# usage: gen-cert.sh <output-dir> <name> <...ip>
# check that we got at least 2 parameters we needed or exit the script with a usage message
[ $# -le 1 ] && { echo "Usage: $0 output-dir name ...ips"; exit 1; }

# give better names to parameter variables
OUT_DIR=$1
NAME=$2
IPS=( "${@:3}" )

pushd "$OUT_DIR" || exit

# generate a key
openssl genrsa -out "${NAME}".key 2048

# write the config file
# shellcheck disable=SC2086
cat > ${NAME}.conf <<EOF

[ req ]
default_bits = 2048
prompt = no
default_md = sha512
req_extensions = v3_req
distinguished_name = dn

[ dn ]
C = DE
L = Berlin
O = OpenFogStack
OU = tinyFaaS
EOF

# write the CN into the config file
echo "CN = ${NAME}" >> "${NAME}".conf

cat >> ${NAME}.conf <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment, digitalSignature
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
EOF

# write the IP SAN into the config file
IPNUM=1

for i in "${IPS[@]}"; do
  IPNUM=$((IPNUM+1))
  echo "IP.${IPNUM} = ${i}" >> "${NAME}".conf
done

# generate the CSR
openssl req -new -key "${NAME}".key -out "${NAME}".csr -config "${NAME}".conf

# build the certificate
openssl x509 -req -in "${NAME}".csr -CA ca.crt -CAkey ca.key \
-CAcreateserial -out "${NAME}".crt -days 1825 \
-extfile "${NAME}".conf -extensions v3_req

# delete the config file and csr
rm "${NAME}".conf
rm "${NAME}".csr

popd || exit
