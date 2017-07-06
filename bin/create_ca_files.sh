#!/bin/bash -ex

KETO_ASSETS_DIR=${KETO_ASSETS_DIR:-${PWD}}

cat <<EOF > ${KETO_ASSETS_DIR}/dummyCSR.json
{
  "CN": "Keto ETCD CA",
  "CA": {
    "expiry": "87600h"
  },
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C": "GB",
      "L": "London",
      "O": "ETCD",
      "OU": "CA",
      "ST": "London"
    }
  ]
}
EOF

function generate_ca() {
  if [[ ! -f ${1}_ca.key ]]; then
    cfssl gencert -initca ${KETO_ASSETS_DIR}/dummyCSR.json | cfssljson -bare ${1}_ca
    mv ${1}_ca.pem ${KETO_ASSETS_DIR}/${1}_ca.crt
    mv ${1}_ca-key.pem ${KETO_ASSETS_DIR}/${1}_ca.key
    rm ${1}_ca.csr
  fi
}

generate_ca etcd
generate_ca kube
