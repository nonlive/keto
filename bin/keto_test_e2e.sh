#!/bin/bash -ex

[[ "${E2E}" = "true" ]] || exit 0

WORKINGDIR="${GOPATH}/src/github.com/UKHomeOffice/keto"

${WORKINGDIR}/bin/keto_test_e2e_setup.sh
${WORKINGDIR}/bin/keto_test_e2e_exec.sh
