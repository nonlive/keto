#!/bin/bash -ex

: ${CLUSTER_NAME:?"[ERROR] Environment Variable 'CLUSTER_NAME' has not been set"}
: ${KETO_CLOUD_PROVIDER:?"[ERROR] Environment Variable 'KETO_CLOUD_PROVIDER' has not been set"}
: ${KETO_SSH_KEY_NAME:?"[ERROR] Environment Variable 'KETO_SSH_KEY_NAME' has not been set"}
: ${TEST_NETWORK_IDS:?"[ERROR] Environment Variable 'TEST_NETWORK_IDS' has not been set"}

set +e

WORKINGDIR="${GOPATH}/src/github.com/UKHomeOffice/keto"
KETO_ASSETS_DIR=${KETO_ASSETS_DIR:-${PWD}}

function generate_kube_config() {
    API_ADDR="https://$(aws cloudformation describe-stacks --stack-name keto-${CLUSTER_NAME}-elb --query "Stacks[0].Outputs[?OutputKey=='ELBDNS'].OutputValue" --output text)"
    mkdir -p ~/.kube/
    kubeadm alpha phase kubeconfig client-certs --client-name kubernetes-admin --organization system:masters --server ${API_ADDR} --cert-dir ${KETO_ASSETS_DIR} > ~/.kube/config
}

function cleanup() {
    echo "[INFO] Attempting to delete keto cluster '${CLUSTER_NAME}'"
    keto --cloud ${KETO_CLOUD_PROVIDER} delete cluster ${CLUSTER_NAME}
}

function run_e2e_test() {
    echo "[INFO] Creating keto cluster with 1 master pool (5 nodes), 1 compute pool (1 node)..."
    keto --cloud ${KETO_CLOUD_PROVIDER} create cluster ${CLUSTER_NAME} \
        --assets-dir ${KETO_ASSETS_DIR} \
        --compute-pools 1 \
        --pool-size 1 \
        --machine-type t2.micro \
        --ssh-key ${KETO_SSH_KEY_NAME} \
        --networks ${TEST_NETWORK_IDS} || return

    # TODO: Modify once Keto adds capability to auto generate config using the client cli
    echo "[INFO] Generating Kubernetes config"
    generate_kube_config || return

    echo "[INFO] Executing kuberang test to validate cluster health"
    ${WORKINGDIR}/bin/kuberang.sh 150 || return

    echo "[INFO] Keto attempting to create compute pool 'compute1'"
    keto --cloud ${KETO_CLOUD_PROVIDER} create computepool compute1 --cluster ${CLUSTER_NAME} \
        --pool-size 3 \
        --machine-type t2.micro \
        --ssh-key ${KETO_SSH_KEY_NAME} \
        --networks ${TEST_NETWORK_IDS} || return

    echo "[INFO] Keto attempting to delete compute pool 'compute0'"
    keto --cloud ${KETO_CLOUD_PROVIDER} delete computepool compute0 --cluster ${CLUSTER_NAME} || return

    echo "[INFO] Executing kuberang test to validate cluster health"
    ${WORKINGDIR}/bin/kuberang.sh 150 || return

    cleanup
}

run_e2e_test

if [ $? -ne 0 ]; then
    cleanup
    echo "[FAIL] Keto end-to-end tests returned a non-zero exit code."
    exit 1
else
    echo "[PASS] Keto & Kuberang executions ran successfully."
    exit 0
fi
