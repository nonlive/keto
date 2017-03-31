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
	"strings"
	"time"

	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	blueStack  = "blue"
	greenStack = "green"

	stackStatusCompleteSuffix   = "COMPLETE"
	stackStatusInProgressSuffix = "IN_PROGRESS"
	stackStatusFailedSuffix     = "FAILED"
	stackStatusRollback         = "ROLLBACK"
)

func (c *Cloud) getClusterInfraStack(clusterName string) (*cloudformation.Stack, error) {
	return c.getStack(makeClusterInfraStackName(clusterName))
}

func (c *Cloud) getNodePoolStack(name, clusterName, part string) (*cloudformation.Stack, error) {
	return c.getStack(makeNodePoolStackName(clusterName, name, part))
}

// stackExists returns true if a given stack name exists and is managed by keto.
func (c *Cloud) stackExists(name string) (bool, error) {
	s, err := c.getStack(name)
	if err != nil || s == nil {
		return false, err
	}
	if isStackManaged(s.Tags) {
		return true, nil
	}
	return false, nil
}

func (c *Cloud) getStack(name string) (*cloudformation.Stack, error) {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(name),
	}
	resp, err := c.cf.DescribeStacks(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" && strings.Contains(awsErr.Message(), "does not exist") {
				return &cloudformation.Stack{}, nil
			}
		}
		return &cloudformation.Stack{}, err
	}
	return resp.Stacks[0], nil
}

// isStackManaged returns true if the given slice of tags contains managed by keto tag.
func isStackManaged(tags []*cloudformation.Tag) bool {
	for _, tag := range tags {
		if *tag.Key == managedByKetoTagKey && *tag.Value == managedByKetoTagValue {
			return true
		}
	}
	return false
}

func (c *Cloud) createClusterInfraStack(clusterName, vpcID string, subnets []*ec2.Subnet) error {
	templateBody, err := renderClusterInfraStackTemplate(clusterName, vpcID, subnets)
	if err != nil {
		return err
	}
	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(makeClusterInfraStackName(clusterName)),
		Tags:         makeStackTags(clusterName),
		TemplateBody: aws.String(templateBody),
	}
	if err := c.createStack(stack); err != nil {
		return err
	}
	return nil
}

// makeClusterInfraStackName returns cluster infra stack name.
// There is no blue/green updates for cluster infra stack. Updates are handled in place.
func makeClusterInfraStackName(clusterName string) string {
	return fmt.Sprintf("keto-%s-infra", clusterName)
}

func (c *Cloud) createMasterStack(p model.MasterPool, infraStackName, amiID, sshKeyPairName string) error {
	templateBody, err := renderMasterStackTemplate(p, infraStackName, amiID, sshKeyPairName)
	if err != nil {
		return err
	}
	stack := &cloudformation.CreateStackInput{
		StackName: aws.String(makeMasterStackName(p.ClusterName, "")),
		Capabilities: aws.StringSlice([]string{
			cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
		Tags:         makeStackTags(p.ClusterName),
		TemplateBody: aws.String(templateBody),
	}
	if err := c.createStack(stack); err != nil {
		return err
	}
	return nil
}

// makeMasterStackName returns master stack name for either blue or green stack.
func makeMasterStackName(clusterName, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-master-%s", clusterName, part)
}

func (c *Cloud) createELBStack(p model.MasterPool, vpcID, infraStackName string) error {
	templateBody, err := renderELBStackTemplate(p, vpcID, infraStackName)
	if err != nil {
		return err
	}
	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(makeELBStackName(p.ClusterName)),
		Tags:         makeStackTags(p.ClusterName),
		TemplateBody: aws.String(templateBody),
	}
	if err := c.createStack(stack); err != nil {
		return err
	}
	return nil
}

// makeELBStackName returns ELB stack name.
// There is no blue/green updates for ELB stack. Updates are handled in place.
func makeELBStackName(clusterName string) string {
	return fmt.Sprintf("keto-%s-elb", clusterName)
}

// makeNodePoolStackName returns nodepool stack name for either blue or green stack.
func makeNodePoolStackName(clusterName, name, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-nodepool-%s-%s", clusterName, name, part)
}

func (c *Cloud) createNodePoolStack() error {
	return nil
}

func makeStackTags(clusterName string) []*cloudformation.Tag {
	tags := []*cloudformation.Tag{
		{
			Key:   aws.String(managedByKetoTagKey),
			Value: aws.String(managedByKetoTagValue),
		},
		{
			Key:   aws.String(clusterNameTagKey),
			Value: aws.String(clusterName),
		},
	}
	return tags
}

// createStack creates a new stack and waits for completion. If stack creation
// fails, an error is returned.
func (c *Cloud) createStack(in *cloudformation.CreateStackInput) error {
	// CloudFormation validation is pretty useless, maybe one day, it'll get better.
	if err := c.validateStackTemplate(in.TemplateBody); err != nil {
		return err
	}

	resp, err := c.cf.CreateStack(in)
	if err != nil {
		return err
	}

	if err := c.waitForStackOperationCompletion(*resp.StackId); err != nil {
		return err
	}
	return nil
}

func (c *Cloud) validateStackTemplate(tpl *string) error {
	params := &cloudformation.ValidateTemplateInput{
		TemplateBody: tpl,
	}
	_, err := c.cf.ValidateTemplate(params)
	if err != nil {
		return err
	}
	return nil
}

// waitForStackOperationCompletion returns an error if a stack
// create/update/delete operation fails. Rollback status also returns an error
// to indicate a failure. Otherwise an error returned is nil.
func (c *Cloud) waitForStackOperationCompletion(id string) error {
	for {
		s, err := c.getStack(id)
		switch {
		case err != nil:
			return err
		// wait for any status that is in progress to complete
		case strings.HasSuffix(*s.StackStatus, stackStatusInProgressSuffix):
			break
		// a failed status is always treated as a failure
		case strings.HasSuffix(*s.StackStatus, stackStatusFailedSuffix):
			return fmt.Errorf("stack operation failed")
		// a rollback status is always treated as a failure
		case strings.Contains(*s.StackStatus, stackStatusRollback):
			return fmt.Errorf("stack operation failed")
		// and finally a complete status is treated as a success
		case strings.HasSuffix(*s.StackStatus, stackStatusCompleteSuffix):
			return nil
		}
		time.Sleep(5 * time.Second)
	}
}
