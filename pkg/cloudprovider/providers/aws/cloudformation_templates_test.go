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
	"testing"

	"github.com/UKHomeOffice/keto/testutil"

	"github.com/UKHomeOffice/keto/pkg/model"

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

	s, err := renderClusterInfraStackTemplate("foo", vpc, subnets)
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, vpc)
}

func TestRenderELBStackTemplate(t *testing.T) {
	pool := model.MasterPool{
		NodePool: model.NodePool{
			ResourceMeta: model.ResourceMeta{ClusterName: "foo"},
			NodePoolSpec: model.NodePoolSpec{Networks: []string{"network0", "network1"}},
		},
	}

	s, err := renderELBStackTemplate(pool, vpc, "infra-foo-stack")
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, vpc)
}

func TestRenderMasterStackTemplate(t *testing.T) {
	pool := model.MasterPool{
		NodePool: model.NodePool{
			ResourceMeta: model.ResourceMeta{ClusterName: "foo"},
			NodePoolSpec: model.NodePoolSpec{Networks: []string{"network0", "network1"}},
		},
	}

	s, err := renderMasterStackTemplate(pool, "infra-foo-stack", ami, "myelb", "assets-bucket")
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

	s, err := renderComputeStackTemplate(pool, "infra-foo-stack", ami)
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, s, ami)
}
