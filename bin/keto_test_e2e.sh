#!/bin/bash -ex

[[ "${E2E}" = "true" ]] || exit 0

working_dir="${GOPATH}/src/github.com/UKHomeOffice/keto"

${working_dir}/bin/keto_test_e2e_setup.sh
${working_dir}/bin/keto_test_e2e_exec.sh
