#!/bin/bash -e

# Duplicate &1 to &5, so we can use tee to write kuberang output to both stdout and var
exec 5>&1

# Poll for API to be available
MAX_RETRIES=${1:-30}
WAIT_TIME=5
RETRIES=0

echo "Waiting for Kube API readiness..."
while true ; do
  PODS_RUNNING=`kubectl --namespace=kube-system get pods 2> /dev/null | grep "kube-dns" | grep "Running" | wc -l`
  if [[ $PODS_RUNNING -eq 0 ]]; then
    RETRIES=$((RETRIES + 1))
    if [[ ${RETRIES} -eq ${MAX_RETRIES} ]]; then
        echo "[ERROR] Max timeout reached while waiting for Kube API to become available."
        exit 1
    else
        echo "[INFO] Attempt #${RETRIES} of #${MAX_RETRIES}: Kube API not yet available, sleeping for ${WAIT_TIME} seconds..."
        sleep ${WAIT_TIME}
    fi
  else
    echo "[INFO] Kube API available and responding, beginning Kuberang test."
    break
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
