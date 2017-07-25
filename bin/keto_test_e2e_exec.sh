#!/bin/bash -ex

: ${CLUSTER_NAME:?"[ERROR] Environment Variable 'CLUSTER_NAME' has not been set"}
: ${KETO_CLOUD_PROVIDER:?"[ERROR] Environment Variable 'KETO_CLOUD_PROVIDER' has not been set"}
: ${KETO_SSH_KEY_NAME:?"[ERROR] Environment Variable 'KETO_SSH_KEY_NAME' has not been set"}
: ${TEST_NETWORK_IDS:?"[ERROR] Environment Variable 'TEST_NETWORK_IDS' has not been set"}
KETO_ASSETS_DIR=${KETO_ASSETS_DIR:-${PWD}}

set +e

working_dir="${GOPATH}/src/github.com/UKHomeOffice/keto"
kube_labels=("customlabel1=customvalue1" "label2=value2")

function generate_kube_config () {
    echo "[INFO] Generating Kubernetes config"
    API_ADDR="https://$(aws cloudformation describe-stacks --stack-name keto-${CLUSTER_NAME}-elb --query "Stacks[0].Outputs[?OutputKey=='ELBDNS'].OutputValue" --output text)"
    mkdir -p ~/.kube/
    kubeadm alpha phase kubeconfig client-certs --client-name kubernetes-admin --organization system:masters --server ${API_ADDR} --cert-dir ${KETO_ASSETS_DIR} > ~/.kube/config
}

function cleanup () {
    echo "[INFO] Attempting to delete keto cluster '${CLUSTER_NAME}'"
    keto --cloud ${KETO_CLOUD_PROVIDER} delete cluster ${CLUSTER_NAME}
}

function check_labels () {
    echo "[INFO] Checking if label $1 is present on $2 nodes"
    label_count=$(kubectl get nodes -l $1 -o json | jq '.items | length')
    if [[ $label_count -ne $2 ]]; then
        echo "[FAIL] Label check failed! Label $1 was found on ${label_count} nodes, expected $2 nodes."
        return 1
    else
        echo "[PASS] Label $1 was present on ${label_count} nodes."
    fi
}

function run_e2e_test () {
    echo "[INFO] Creating keto cluster with 1 master pool (5 nodes), 1 compute pool (1 node)..."
    keto --cloud ${KETO_CLOUD_PROVIDER} create cluster ${CLUSTER_NAME} \
        --assets-dir ${KETO_ASSETS_DIR} \
        --compute-pools 1 \
        --pool-size 1 \
        --machine-type t2.micro \
        --ssh-key ${KETO_SSH_KEY_NAME} \
        --networks ${TEST_NETWORK_IDS} \
        --labels ${kube_labels[0]} || return

    # TODO: Modify on approval of Keto PR #99
    generate_kube_config || return

    ${working_dir}/bin/kuberang.sh 180 || return

    check_labels ${kube_labels[0]} 6 || return

    echo "[INFO] Keto attempting to create compute pool 'compute1' with 2 nodes..."
    keto --cloud ${KETO_CLOUD_PROVIDER} create computepool compute1 --cluster ${CLUSTER_NAME} \
        --pool-size 2 \
        --machine-type t2.micro \
        --ssh-key ${KETO_SSH_KEY_NAME} \
        --networks ${TEST_NETWORK_IDS} \
        --labels ${kube_labels[1]} || return

    echo "[INFO] Keto attempting to delete compute pool 'compute0'"
    keto --cloud ${KETO_CLOUD_PROVIDER} delete computepool compute0 --cluster ${CLUSTER_NAME} || return

    ${working_dir}/bin/kuberang.sh 180 || return

    check_labels ${kube_labels[0]} 7 || return
    check_labels ${kube_labels[1]} 2 || return

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
