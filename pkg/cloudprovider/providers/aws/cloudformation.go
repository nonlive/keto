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
	"strconv"
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

	stackTypeTagKey        = "stack-type"
	poolNameTagKey         = "pool-name"
	coreOSVersionTagKey    = "coreos-version"
	kubeVersionTagKey      = "kube-version"
	kubeAPIURLTagKey       = "kube-api-url"
	machineTypeTagKey      = "machine-type"
	diskSizeTagKey         = "disk-size"
	assetsBucketNameTagKey = "assets-bucket-name"

	clusterInfraStackType = "infra"
	elbStackType          = "elb"
	masterPoolStackType   = "masterpool"
	computePoolStackType  = "computepool"

	stackStatusCompleteSuffix   = "COMPLETE"
	stackStatusInProgressSuffix = "IN_PROGRESS"
	stackStatusFailedSuffix     = "FAILED"
	stackStatusRollback         = "ROLLBACK"
)

var (
	// List of tag key names that should not be used as cluster or pool labels
	stackTagsNotLabels = []string{
		stackTypeTagKey,
		managedByKetoTagKey,
		kubeVersionTagKey,
		kubeAPIURLTagKey,
		coreOSVersionTagKey,
		assetsBucketNameTagKey,
	}
)

// stackExists returns true if a given stack name exists and is managed by keto.
func (c *Cloud) stackExists(name string) (bool, error) {
	s, err := c.getStack(name)
	if err != nil || s == nil {
		return false, err
	}
	if isStackManaged(s) {
		return true, nil
	}
	return false, nil
}

// getStack returns Stack struct given stack name and an error, if any.
func (c *Cloud) getStack(name string) (*cloudformation.Stack, error) {
	stacks, err := c.describeStacks(name)
	if err != nil || len(stacks) != 1 {
		return &cloudformation.Stack{}, err
	}
	return stacks[0], nil
}

// getStackLabels returns a model.Labels of given a cloudformation stack
func getStackLabels(s *cloudformation.Stack) model.Labels {
	reservedTag := func(t *cloudformation.Tag) bool {
		for _, i := range stackTagsNotLabels {
			if *t.Key == i {
				return true
			}
		}
		return false
	}

	labels := model.Labels{}
	for _, tag := range s.Tags {
		if reservedTag(tag) {
			continue
		}
		labels[*tag.Key] = *tag.Value
	}
	return labels
}

// getStacksByType returns a list of stacks by type, also checks if they are
// managed by keto. An error is returned as well, if any.
func (c *Cloud) getStacksByType(t string) ([]*cloudformation.Stack, error) {
	allStacks, err := c.describeStacks("")
	stacks := []*cloudformation.Stack{}
	if err != nil {
		return stacks, err
	}

	for _, s := range allStacks {
		// Skip over stacks that are not managed by keto
		if !isStackManaged(s) {
			continue
		}

		for _, tag := range s.Tags {
			if *tag.Key == stackTypeTagKey && *tag.Value == t {
				stacks = append(stacks, s)
			}
		}
	}
	return stacks, err
}

// getStackResources returns a list of stack resources given a stack name.
func (c *Cloud) getStackResources(name string) ([]*cloudformation.StackResource, error) {
	params := &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(name),
	}

	resp, err := c.cf.DescribeStackResources(params)
	if err != nil {
		return []*cloudformation.StackResource{}, err
	}
	return resp.StackResources, nil
}

// describeStacks runs a DescribeStacks on all stacks or a particular stack specified
// by name. It returns a slice of cloudformation stacks and an error if any.
func (c *Cloud) describeStacks(name string) ([]*cloudformation.Stack, error) {
	params := &cloudformation.DescribeStacksInput{}
	if name != "" {
		params = params.SetStackName(name)
	}
	resp, err := c.cf.DescribeStacks(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" && strings.Contains(awsErr.Message(), "does not exist") {
				return []*cloudformation.Stack{}, nil
			}
		}
		return []*cloudformation.Stack{}, err
	}
	return resp.Stacks, nil
}

// isStackManaged returns true if the given stack tags contains managed by keto tag.
func isStackManaged(s *cloudformation.Stack) bool {
	for _, tag := range s.Tags {
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
		Tags:         makeStackTags(clusterName, clusterInfraStackType, "", "", "", "", "", "", 0),
		TemplateBody: aws.String(templateBody),
	}

	return c.createStack(stack)
}

// makeClusterInfraStackName returns cluster infra stack name.
// There is no blue/green updates for cluster infra stack. Updates are handled in place.
func makeClusterInfraStackName(clusterName string) string {
	return fmt.Sprintf("keto-%s-%s", clusterName, clusterInfraStackType)
}

func (c *Cloud) createMasterPoolStack(
	p model.MasterPool,
	infraStackName string,
	amiID string,
	elbName string,
	kubeAPIURL string,
	assetsBucketName string,
) error {

	templateBody, err := renderMasterStackTemplate(p, infraStackName, amiID, elbName, assetsBucketName)
	if err != nil {
		return err
	}
	stack := &cloudformation.CreateStackInput{
		StackName: aws.String(makeMasterPoolStackName(p.ClusterName, "")),
		Capabilities: aws.StringSlice([]string{
			cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
		Tags: makeStackTags(p.ClusterName, masterPoolStackType, p.Name, p.KubeVersion, p.CoreOSVersion,
			p.MachineType, kubeAPIURL, assetsBucketName, p.DiskSize),
		TemplateBody: aws.String(templateBody),
	}

	return c.createStack(stack)
}

// makeMasterPoolStackName returns master stack name for either blue or green stack.
func makeMasterPoolStackName(clusterName, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-%s-%s", clusterName, masterPoolStackType, part)
}

func (c *Cloud) createELBStack(p model.MasterPool, vpcID, infraStackName string) error {
	templateBody, err := renderELBStackTemplate(p, vpcID, infraStackName)
	if err != nil {
		return err
	}
	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(makeELBStackName(p.ClusterName)),
		Tags:         makeStackTags(p.ClusterName, elbStackType, "", "", "", "", "", "", 0),
		TemplateBody: aws.String(templateBody),
	}

	return c.createStack(stack)
}

// makeELBStackName returns ELB stack name.
// There is no blue/green updates for ELB stack. Updates are handled in place.
func makeELBStackName(clusterName string) string {
	return fmt.Sprintf("keto-%s-%s", clusterName, elbStackType)
}

// makeComputePoolStackName returns compute pool stack name for either blue or
// green stack.
func makeComputePoolStackName(clusterName, name, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-%s-%s", clusterName, name, part)
}

func (c *Cloud) createNodePoolStack() error {
	return ErrNotImplemented
}

// makeStackTags returns a list of cloudformation tags that are applied to all stacks.
func makeStackTags(
	clusterName string,
	stackType string,
	poolName string,
	kubeVersion string,
	osVersion string,
	machineType string,
	kubeAPIURL string,
	assetsBucketName string,
	diskSize int,
) []*cloudformation.Tag {
	tags := []*cloudformation.Tag{
		{
			Key:   aws.String(managedByKetoTagKey),
			Value: aws.String(managedByKetoTagValue),
		},
		{
			Key:   aws.String(clusterNameTagKey),
			Value: aws.String(clusterName),
		},
		{
			Key:   aws.String(stackTypeTagKey),
			Value: aws.String(stackType),
		},
	}

	if poolName != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(poolNameTagKey),
			Value: aws.String(poolName),
		})
	}
	if kubeVersion != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(kubeVersionTagKey),
			Value: aws.String(kubeVersion),
		})
	}
	if osVersion != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(coreOSVersionTagKey),
			Value: aws.String(osVersion),
		})
	}
	if machineType != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(machineTypeTagKey),
			Value: aws.String(machineType),
		})
	}
	if kubeAPIURL != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(kubeAPIURLTagKey),
			Value: aws.String(kubeAPIURL),
		})
	}
	if assetsBucketName != "" {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(assetsBucketNameTagKey),
			Value: aws.String(assetsBucketName),
		})
	}
	if diskSize > 0 {
		tags = append(tags, &cloudformation.Tag{
			Key:   aws.String(diskSizeTagKey),
			Value: aws.String(strconv.Itoa(diskSize)),
		})
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
	return err
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
