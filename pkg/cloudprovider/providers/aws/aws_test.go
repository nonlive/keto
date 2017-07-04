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
//go:generate mockery -dir $GOPATH/src/github.com/UKHomeOffice/keto/vendor/github.com/aws/aws-sdk-go/service/elb/elbiface -name=ELBAPI
//go:generate mockery -dir $GOPATH/src/github.com/UKHomeOffice/keto/vendor/github.com/aws/aws-sdk-go/service/route53/route53iface -name=Route53API

package aws

import (
	"log"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider/providers/aws/mocks"
	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/UKHomeOffice/keto/testutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"

	"github.com/stretchr/testify/mock"
)

func makeLogger() *log.Logger {
	return log.New(os.Stderr, "", log.LstdFlags)
}

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
	mockEC2 := &mocks.EC2API{}
	mockR53 := &mocks.Route53API{}
	c := &Cloud{
		Logger: makeLogger(),
		cf:     mockCF,
		ec2:    mockEC2,
		r53:    mockR53,
	}

	clusterName := "foo"
	p := model.MasterPool{NodePool: testutil.MakeNodePool(clusterName, "master")}
	p.Networks = []string{"network0", "network1"}
	cluster := model.Cluster{
		ResourceMeta: model.ResourceMeta{Name: clusterName},
		MasterPool:   p,
		DNSZone:      "dnszone.local",
	}

	returnSubnetsFunc := func() *ec2.DescribeSubnetsOutput {
		subnets := []*ec2.Subnet{}
		for _, n := range p.Networks {
			subnets = append(subnets, &ec2.Subnet{
				SubnetId:         aws.String(n),
				VpcId:            aws.String("vpc0"),
				AvailabilityZone: aws.String("az0"),
			})
		}
		return &ec2.DescribeSubnetsOutput{Subnets: subnets}
	}

	mockR53.On("ListHostedZonesByName", &route53.ListHostedZonesByNameInput{
		DNSName: aws.String(cluster.DNSZone),
	}).Return(&route53.ListHostedZonesByNameOutput{HostedZones: []*route53.HostedZone{{Name: aws.String(cluster.DNSZone)}}}, nil)

	mockEC2.On("DescribeSubnets", &ec2.DescribeSubnetsInput{
		SubnetIds: aws.StringSlice(p.Networks),
	}).Return(returnSubnetsFunc(), nil)

	mockCF.On("ValidateTemplate", mock.AnythingOfType("*cloudformation.ValidateTemplateInput")).Return(
		&cloudformation.ValidateTemplateOutput{}, nil)

	stackName := makeClusterInfraStackName(clusterName)
	mockCF.On("CreateStack", mock.AnythingOfType("*cloudformation.CreateStackInput")).Return(
		&cloudformation.CreateStackOutput{StackId: aws.String(stackName)}, nil)

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(stackName)}).Return(
		&cloudformation.DescribeStacksOutput{
			Stacks: []*cloudformation.Stack{
				{
					StackId:     aws.String(stackName),
					StackStatus: aws.String(cloudformation.StackStatusCreateComplete),
				},
			},
		}, nil)

	if err := c.CreateClusterInfra(cluster); err != nil {
		t.Error(err)
	}

	mockCF.AssertExpectations(t)
}

func TestGetClusters(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		Logger: makeLogger(),
		cf:     mockCF,
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
				{
					Key:   aws.String(internalClusterTagKey),
					Value: aws.String("true"),
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
	if !res[0].Internal {
		t.Errorf("failed to read cluster Internal flag, got %v; want %v", res[0].Internal, true)
	}

	mockCF.AssertExpectations(t)
}

func TestDeleteComputePool(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		Logger: makeLogger(),
		cf:     mockCF,
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

func TestCreateMasterPool(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	mockEC2 := &mocks.EC2API{}
	mockELB := &mocks.ELBAPI{}
	c := &Cloud{
		Logger: makeLogger(),
		cf:     mockCF,
		ec2:    mockEC2,
		elb:    mockELB,
	}

	clusterName := "foo"
	infraSubnet := "infranetwork0"
	p := model.MasterPool{NodePool: testutil.MakeNodePool(clusterName, "master")}

	mockEC2.On("DescribeNetworkInterfaces", &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:" + managedByKetoTagKey),
				Values: []*string{aws.String(managedByKetoTagValue)},
			},
			{
				Name:   aws.String("tag:" + clusterNameTagKey),
				Values: []*string{aws.String(clusterName)},
			},
		},
	}).Return(&ec2.DescribeNetworkInterfacesOutput{
		NetworkInterfaces: []*ec2.NetworkInterface{
			{
				SubnetId: aws.String(infraSubnet),
			},
		},
	}, nil).Once()

	mockEC2.On("DescribeImages", &ec2.DescribeImagesInput{
		Owners: []*string{aws.String(coreOSAWSAccountID)},
		Filters: []*ec2.Filter{
			{Name: aws.String("name"), Values: []*string{aws.String(p.CoreOSVersion)}},
			{Name: aws.String("virtualization-type"), Values: []*string{aws.String("hvm")}},
			{Name: aws.String("state"), Values: []*string{aws.String("available")}},
		},
	}).Return(&ec2.DescribeImagesOutput{
		Images: []*ec2.Image{
			{
				ImageId: aws.String("ami123"),
			},
		},
	}, nil).Once()

	mockCF.On("DescribeStackResources", &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(makeELBStackName(clusterName)),
	}).Return(&cloudformation.DescribeStackResourcesOutput{
		StackResources: []*cloudformation.StackResource{
			{
				ResourceType:       aws.String("AWS::ElasticLoadBalancing::LoadBalancer"),
				PhysicalResourceId: aws.String("elb-physical-id"),
			},
		},
	}, nil)

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(makeELBStackName(p.ClusterName))}).Return(
		&cloudformation.DescribeStacksOutput{
			Stacks: []*cloudformation.Stack{
				{
					StackId: aws.String("elb-stack-id"),
					Outputs: []*cloudformation.Output{
						{
							OutputKey:   aws.String("ELBDNS"),
							OutputValue: aws.String("kube-" + p.ClusterName),
						},
					},
				},
			},
		}, nil).Once()

	mockCF.On("DescribeStackResources", &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(makeClusterInfraStackName(clusterName)),
	}).Return(&cloudformation.DescribeStackResourcesOutput{
		StackResources: []*cloudformation.StackResource{
			{
				ResourceType:       aws.String("AWS::S3::Bucket"),
				PhysicalResourceId: aws.String("s3-assets-bucket"),
			},
		},
	}, nil).Once()

	mockEC2.On("DescribeSubnets", &ec2.DescribeSubnetsInput{
		SubnetIds: aws.StringSlice([]string{infraSubnet})}).Return(&ec2.DescribeSubnetsOutput{
		Subnets: []*ec2.Subnet{
			{
				SubnetId:         aws.String(infraSubnet),
				VpcId:            aws.String("vpc0"),
				AvailabilityZone: aws.String("az0"),
			}}},
		nil)

	mockCF.On("ValidateTemplate", mock.AnythingOfType("*cloudformation.ValidateTemplateInput")).Return(
		&cloudformation.ValidateTemplateOutput{}, nil)

	masterPoolStackID := "masterpool-stack-id"
	mockCF.On("CreateStack", mock.MatchedBy(func(in *cloudformation.CreateStackInput) bool {
		// Assume that the cluster infra already exists in a single AZ, so the
		// specified subnets, when creating this MasterPool, must be ignored and
		// cluster infra subnets must be used.
		return strings.Contains(*in.TemplateBody, infraSubnet)
	})).Return(
		&cloudformation.CreateStackOutput{
			StackId: aws.String(masterPoolStackID),
		}, nil)

	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(masterPoolStackID)}).Return(
		&cloudformation.DescribeStacksOutput{
			Stacks: []*cloudformation.Stack{
				{
					StackId:     aws.String(masterPoolStackID),
					StackStatus: aws.String(cloudformation.StackStatusDeleteComplete),
				},
			},
		}, nil)

	if err := c.CreateMasterPool(p); err != nil {
		t.Error(err)
	}

	mockCF.AssertExpectations(t)
	mockELB.AssertExpectations(t)
	mockEC2.AssertExpectations(t)
}

func TestGetKubeAPIURL(t *testing.T) {
	mockCF := &mocks.CloudFormationAPI{}
	c := &Cloud{
		Logger: makeLogger(),
		cf:     mockCF,
	}

	clusterName := "foo"
	mockCF.On("DescribeStacks", &cloudformation.DescribeStacksInput{StackName: aws.String(makeELBStackName(clusterName))}).Return(
		&cloudformation.DescribeStacksOutput{
			Stacks: []*cloudformation.Stack{
				{
					StackId: aws.String("elb-stack-id"),
					Outputs: []*cloudformation.Output{
						{
							OutputKey:   aws.String("ELBDNS"),
							OutputValue: aws.String("kube-" + clusterName + ".local"),
						},
					},
				},
			},
		}, nil).Once()

	result, err := c.getKubeAPIURL(clusterName)
	if err != nil {
		t.Error(err)
	}
	u, err := url.Parse(result)
	if err != nil {
		t.Errorf("failed to parse URL, got %q", u)
	}
	if u.Scheme != "https" {
		t.Errorf("url scheme is not correct, got %q; wanted %q", u.Scheme, "https")
	}

	mockCF.AssertExpectations(t)
}
