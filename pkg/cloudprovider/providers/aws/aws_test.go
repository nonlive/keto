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

//go:generate mockery -dir $GOPATH/src/github.com/UKHomeOffice/keto/vendor/github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface -name=CloudFormationAPI
//go:generate mockery -dir $GOPATH/src/github.com/UKHomeOffice/keto/vendor/github.com/aws/aws-sdk-go/service/ec2/ec2iface -name=EC2API

package aws

import (
	"testing"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider/providers/aws/mocks"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func TestGetVPCIDFromSubnetList(t *testing.T) {
	testCases := []struct {
		name  string
		input []*ec2.Subnet
		want  string
	}{
		{
			"nil input",
			nil,
			"",
		},
		{
			"vpc123 valid",
			[]*ec2.Subnet{
				{VpcId: aws.String("vpc123")},
				{VpcId: aws.String("vpc123")},
				{VpcId: aws.String("vpc123")},
			},
			"vpc123",
		},
		{
			"not all subnets in the same VPC",
			[]*ec2.Subnet{
				{VpcId: aws.String("vpc123")},
				{VpcId: aws.String("vpc123")},
				{VpcId: aws.String("vpc000")},
			},
			"",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			got, err := getVpcIDFromSubnetList(c.input)
			if got != c.want {
				t.Errorf("got %q; want %q; err %v", got, c.want, err)
			}

		})
	}
}

func TestCreateClusterInfra(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		cf: mockCF,
	}

	infraStackName := "keto-foo-infra"
	stacks := []*cloudformation.Stack{
		{
			StackName: aws.String(infraStackName),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(clusterInfraStackType),
				},
			},
		},
	}

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(infraStackName)}).Return(
		&cloudformation.DescribeStacksOutput{Stacks: stacks}, nil)

	cluster := model.Cluster{Name: "foo"}
	err := c.CreateClusterInfra(cluster)
	if err == nil {
		t.Error("should return an error")
	}

	mockCF.AssertExpectations(t)
}

func TestGetClusters(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		cf: mockCF,
	}

	stacks := []*cloudformation.Stack{
		{
			StackName: aws.String("keto-foo-infra"),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(clusterNameTagKey),
					Value: aws.String("foo"),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(clusterInfraStackType),
				},
			},
		},
		{
			StackName: aws.String("keto-bar-infra"),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(clusterNameTagKey),
					Value: aws.String("bar"),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(clusterInfraStackType),
				},
			},
		},
	}

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{}).Return(
		&cloudformation.DescribeStacksOutput{Stacks: stacks}, nil)

	res, err := c.GetClusters("foo")
	if err != nil {
		t.Error(err)
	}
	if len(res) != 1 {
		t.Fatalf("should have received one result, but got %d instead", len(res))
	}
	if res[0].Name != "foo" {
		t.Errorf("got wrong cluster %q", res[0].Name)
	}

	mockCF.AssertExpectations(t)
}

func TestDeleteComputePool(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		cf: mockCF,
	}

	stacks := []*cloudformation.Stack{
		{
			StackId:     aws.String("foo-id"),
			StackName:   aws.String("keto-foo-compute"),
			StackStatus: aws.String(cloudformation.StackStatusDeleteComplete),
			Tags: []*cloudformation.Tag{
				{
					Key:   aws.String(managedByKetoTagKey),
					Value: aws.String(managedByKetoTagValue),
				},
				{
					Key:   aws.String(clusterNameTagKey),
					Value: aws.String("foo"),
				},
				{
					Key:   aws.String(poolNameTagKey),
					Value: aws.String("compute"),
				},
				{
					Key:   aws.String(stackTypeTagKey),
					Value: aws.String(computePoolStackType),
				},
			},
		},
	}

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{}).Return(
		&cloudformation.DescribeStacksOutput{Stacks: stacks}, nil)

	mockCF.On("DeleteStack", &cloudformation.DeleteStackInput{StackName: aws.String("foo-id")}).Return(
		&cloudformation.DeleteStackOutput{}, nil)

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String("foo-id")}).Return(
		&cloudformation.DescribeStacksOutput{Stacks: stacks}, nil)

	if err := c.DeleteComputePool("foo", "compute"); err != nil {
		t.Error(err)
	}

	mockCF.AssertExpectations(t)
}
