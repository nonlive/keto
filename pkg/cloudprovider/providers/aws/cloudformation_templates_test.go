/*
Copyright 2017 The Keto Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package aws

import (
	"fmt"
	"testing"

	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/UKHomeOffice/keto/testutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	vpc = "vpc123"
	ami = "ami123"
)

func TestRenderClusterInfraStackTemplate(t *testing.T) {
	subnets := []*ec2.Subnet{
		{
			SubnetId:         aws.String("subnet0"),
			AvailabilityZone: aws.String("az0"),
		},
		{
			SubnetId:         aws.String("subnet1"),
			AvailabilityZone: aws.String("az1"),
		},
	}

	networks := getNodesDistributionAcrossNetworks(subnets)
	cluster := model.Cluster{
		ResourceMeta: model.ResourceMeta{
			Name:     "foo",
			Internal: false,
		},
	}

	s, err := renderClusterInfraStackTemplate(cluster, vpc, networks)
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, vpc)
}

func TestRenderELBStackTemplate(t *testing.T) {
	c := model.Cluster{
		MasterPool: model.MasterPool{
			NodePool: model.NodePool{
				ResourceMeta: model.ResourceMeta{ClusterName: "foo"},
				NodePoolSpec: model.NodePoolSpec{Networks: []string{"network0", "network1"}},
			},
		},
	}

	s, err := renderELBStackTemplate(c, vpc)
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, vpc)
}

func TestRenderMasterStackTemplate(t *testing.T) {
	nodesPerSubnet := map[string]int{
		"network0": 2,
		"network1": 2,
		"network2": 1,
	}

	pool := model.MasterPool{
		NodePool: model.NodePool{
			ResourceMeta: model.ResourceMeta{ClusterName: "foo"},
			NodePoolSpec: model.NodePoolSpec{Networks: []string{"network0", "network1"}},
		},
	}

	s, err := renderMasterStackTemplate(pool, ami, "myelb", "assets-bucket", nodesPerSubnet, "https://kube", "mystack")
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, ami)
}

func TestRenderComputeStackTemplate(t *testing.T) {
	pool := model.ComputePool{
		NodePool: model.NodePool{
			ResourceMeta: model.ResourceMeta{ClusterName: "foo"},
			NodePoolSpec: model.NodePoolSpec{Networks: []string{"network0", "network1"}},
		},
	}

	s, err := renderComputeStackTemplate(pool, "infra-foo-stack", ami, "mystack")
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, ami)
}

func TestGetNodesDistribution(t *testing.T) {
	cases := [][]*ec2.Subnet{
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az1")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n2"), AvailabilityZone: aws.String("az2")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n2"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n3"), AvailabilityZone: aws.String("az1")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n2"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n3"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n4"), AvailabilityZone: aws.String("az1")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n2"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n3"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n4"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n5"), AvailabilityZone: aws.String("az2")},
		},
		[]*ec2.Subnet{
			{SubnetId: aws.String("n1"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n2"), AvailabilityZone: aws.String("az0")},
			{SubnetId: aws.String("n5"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n4"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n6"), AvailabilityZone: aws.String("az1")},
			{SubnetId: aws.String("n3"), AvailabilityZone: aws.String("az2")},
			{SubnetId: aws.String("n0"), AvailabilityZone: aws.String("az0")},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%d networks", len(c)), func(t *testing.T) {
			dist := getNodesDistributionAcrossNetworks(c)
			if dist[0].Subnet != "n0" {
				t.Error("subnets do not appear to be sorted")
			}
			if len(dist) == 0 {
				t.Error("there cannot be zero nodes per network")
			}
			if len(dist)%2 == 0 {
				t.Error("got an even number of nodes in total")
			}
			if len(c) > 1 && len(dist) < 5 {
				t.Error("there must be at least 5 nodes in total for >1 networks")
			}
		})
	}
}
