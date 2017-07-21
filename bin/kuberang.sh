#!/bin/bash -e

# Duplicate &1 to &5, so we can use tee to write kuberang output to both stdout and var
exec 5>&1

# Poll for API to be available
MAX_RETRIES=${1:-30}
WAIT_TIME=5
RETRIES=0

echo "Waiting for Kube API readiness..."

PREP_STATE="Checking Kube API availability."
while true ; do
  if kubectl version &> /dev/null; then
    PREP_STATE="Kube API is ready, checking 'kube-dns' pod is running on compute node."

    PODS_RUNNING=`kubectl --namespace=kube-system get pods 2> /dev/null | grep "kube-dns" | grep "Running" | wc -l`
    if [[ $PODS_RUNNING -gt 0 ]]; then
      echo "[INFO] Kube API and compute nodes are available, beginning Kuberang test."
      break
    fi
  fi

  RETRIES=$((RETRIES + 1))
  if [[ ${RETRIES} -eq ${MAX_RETRIES} ]]; then
      echo "[ERROR] Max timeout reached. Failed on step: ${PREP_STATE}"
      exit 1
  else
      echo "[INFO] Attempt #${RETRIES} of #${MAX_RETRIES}: Kube API not yet available, sleeping for ${WAIT_TIME} seconds..."
      sleep ${WAIT_TIME}
  fi
done

# Execute kuberang, writing results to stdout and variable
kuberangResults=$(kuberang | tee >(cat - >&5))

# Check for ERROR in results, exit -1 if present
if [ "$kuberangResults" != "${kuberangResults#*\[ERROR\]}" ]; then
  echo "[FAIL] Kuberang run failed, check output logs for more detailed results."
  exit 1
else
  echo "[PASS] Kuberang run passed successfully."
  exit 0
fi
