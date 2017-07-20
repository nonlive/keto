#!/bin/bash -ex

WORKINGDIR="${GOPATH}/src/github.com/UKHomeOffice/keto"
KUBE_VERSION=`grep -oP 'DefaultKubeVersion\s*=\s*"\K.*?(?=")' ${WORKINGDIR}/pkg/constants/constants.go`
KETO_ASSETS_DIR=${KETO_ASSETS_DIR:-${PWD}}

mkdir -p /go/bin

# Get keto dependencies
curl -s https://glide.sh/get | sh
glide install

# cfssl for cert generation
go get -u github.com/cloudflare/cfssl/cmd/...

# TODO: This can be removed when keto implements the capability to locally generate the kube config
curl -LO https://bootstrap.pypa.io/get-pip.py && python get-pip.py && pip install awscli

# Get kubectl, kubeadm, kuberang
curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl
chmod +x kubectl && mv kubectl /usr/local/bin/kubectl && kubectl help

curl -LO https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubeadm
chmod +x kubeadm && mv kubeadm /usr/local/bin/kubeadm && kubeadm version

curl -LO https://kismatic-installer.s3-accelerate.amazonaws.com/kuberang/latest/kuberang-linux-amd64
chmod +x kuberang-linux-amd64 && mv kuberang-linux-amd64 /usr/local/bin/kuberang

# Generate assets (cert files) required for cluster build and kube config generation
mkdir -p ${KETO_ASSETS_DIR}
${WORKINGDIR}/bin/create_ca_files.sh
ln -s ${KETO_ASSETS_DIR}/kube_ca.crt ${KETO_ASSETS_DIR}/ca.crt
ln -s ${KETO_ASSETS_DIR}/kube_ca.key ${KETO_ASSETS_DIR}/ca.key

# Build Keto
go install -a -v github.com/${DRONE_REPO}/cmd/keto
