# keto

Keto is under development and should not be used or treated as working
software. Everything will probably change.

## Pre-requisites

### Install golang

Keto is built against the latest golang which can be installed from https://golang.org/dl/.

### Building keto

Get the source code:
```
$ go get -u github.com/UKHomeOffice/keto/cmd/keto
```

First install Glide - https://github.com/Masterminds/glide#install.

Get the correct dependency versions:
```
cd ${GOPATH}/src/github.com/UKHomeOffice/keto
glide install
```

Build the binary:
```
go build -v github.com/UKHomeOffice/keto/cmd/keto
```

### Cloud resources

### AWS

You will need the following AWS resources created in advance:

1. An existing VPC
2. Subnet(s) A minimum of one subnet is required
3. An AWS defined EC2 "keypair" ssh-key

## Usage

### Help
```
keto --help
```

### Create Cluster

You will need to [create](#create-expected-ca-files) or obtain suitable CA certs before running keto.
Minimal command to create a Kubernetes cluster in AWS:
```
keto create cluster testcluster --ssh-key my-aws-key-name --networks subnet-awsid --machine-type t2.medium --cloud aws
```

This will create a cluster and an ELB serving the Kubernetes API.

### List Clusters
```
keto get cluster --cloud aws
```

### Delete a cluster
```
keto delete cluster --name testcluster --cloud aws
```

## Create Expected CA Files

1. Retrieve the prerequisite libraries: `go get -u github.com/cloudflare/cfssl/cmd/...`
2. Set an environment variable for the keto assets directory `KETO_ASSETS_DIR` (defaults to `${PWD}`)
3. Create the required ca key and crt files (etcd, kube): `./bin/create_ca_files.sh`


## Run End to End Tests

The e2e tests can be run in CI using the following command:
```
drone deploy -p E2E=true UKHomeOffice/keto <build-number> e2e
```

The following environment variables must be set for the e2e tests to execute successfully:
- CLUSTER_NAME: The name of the Keto cluster you intend to build and test
- KETO_CLOUD_PROVIDER: The cloud provider to run the tests against
- KETO_SSH_KEY_NAME: The name of the ssh key to be used (must already exist within the cloud provider)
- TEST_NETWORK_IDS: A comma separated list of ids (subnets) to build the infrastructure within (must already exist within the cloud provider)

