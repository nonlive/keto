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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"testing"
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
