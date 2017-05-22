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
Create config for `cfssl`:
```
cat <<EOF > config.json 
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
```

Create the required ca key and crt files:
```
function generate_ca() {
  if [[ ! -f ${1}_ca.key ]]; then
    cfssl gencert -initca config.json | cfssljson -bare ${1}_ca
    mv ${1}_ca.pem ${1}_ca.crt
    mv ${1}_ca-key.pem ${1}_ca.key
    rm ${1}_ca.csr
  fi
}

generate_ca etcd
generate_ca kube
```