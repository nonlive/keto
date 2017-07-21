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
	"sort"
	"strings"
	"time"

	"github.com/UKHomeOffice/keto/pkg/keto/util"
	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	blueStack  = "blue"
	greenStack = "green"

	// Stack Outputs key names.
	stackTypeOutputKey                  = "StackType"
	clusterNameOutputKey                = "ClusterName"
	poolNameOutputKey                   = "PoolName"
	coreOSVersionOutputKey              = "CoreOSVersion"
	kubeVersionOutputKey                = "KubeVersion"
	kubeAPIURLOutputKey                 = "KubeAPIURL"
	machineTypeOutputKey                = "MachineType"
	diskSizeOutputKey                   = "DiskSize"
	assetsBucketNameOutputKey           = "AssetsBucketName"
	internalClusterOutputKey            = "InternalCluster"
	labelsOutputKey                     = "Labels"
	elbDNSOutputKey                     = "ELBDNS"
	taintsOutputKey                     = "Taints"
	kubeletExtraArgsOutputKey           = "KubeletExtraArgs"
	apiServerExtraArgsOutputKey         = "APIServerExtraArgs"
	controllerManagerExtraArgsOutputKey = "ControllerManagerExtraArgs"
	schedulerExtraArgsOutputKey         = "SchedulerExtraArgs"

	clusterInfraStackType = "infra"
	elbStackType          = "elb"
	masterPoolStackType   = "masterpool"
	computePoolStackType  = "computepool"

	stackStatusCompleteSuffix   = "COMPLETE"
	stackStatusInProgressSuffix = "IN_PROGRESS"
	stackStatusFailedSuffix     = "FAILED"
	stackStatusRollback         = "ROLLBACK"
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

// getStackLabels returns model.Labels of given cloudformation stack.
func getStackLabels(s *cloudformation.Stack) model.Labels {
	var labels model.Labels
	for _, o := range s.Outputs {
		if *o.OutputKey == labelsOutputKey {
			s := strings.Split(*o.OutputValue, ",")
			labels = util.KVsToStringMap(s)
		}
	}
	return labels
}

// getStackTaints returns model.Taints of given cloudformation stack.
func getStackTaints(s *cloudformation.Stack) model.Taints {
	var taints model.Taints
	for _, o := range s.Outputs {
		if *o.OutputKey == taintsOutputKey {
			s := strings.Split(*o.OutputValue, ",")
			taints = util.KVsToStringMap(s)
		}
	}
	return taints
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

		for _, o := range s.Outputs {
			if *o.OutputKey == stackTypeOutputKey && *o.OutputValue == t {
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
		params.StackName = aws.String(name)
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

func (c *Cloud) createClusterInfraStack(cluster model.Cluster, vpcID string, subnets []*ec2.Subnet) error {
	networks := getNodesDistributionAcrossNetworks(subnets)

	templateBody, err := renderClusterInfraStackTemplate(cluster, vpcID, networks)
	if err != nil {
		return err
	}

	// To ensure stack resources inherit cluster-name.
	tags := make(map[string]string)
	tags[clusterNameTagKey] = cluster.Name
	tags[stackTypeTagKey] = clusterInfraStackType

	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(makeClusterInfraStackName(cluster.Name)),
		Tags:         makeStackTags(tags),
		TemplateBody: aws.String(templateBody),
	}

	return c.createStack(stack)
}

type nodesNetwork struct {
	Subnet           string
	AvailabilityZone string
	NodeID           int
}

// getNodesDistributionAcrossNetworks calculates a number of nodes per network,
// given a list of networks. Single network setup gets 3 nodes. Multi-network
// setup get at least 5 nodes or more. Returns a list of nodesNetwork.
func getNodesDistributionAcrossNetworks(subnets []*ec2.Subnet) []nodesNetwork {
	// Make sure ec2 subnets are sorted by SubnetId.
	sort.Slice(subnets, func(i, j int) bool { return *subnets[i].SubnetId < *subnets[j].SubnetId })

	dist := []nodesNetwork{}

	if len(subnets) == 1 {
		for i := 0; i < 3; i++ {
			dist = append(dist, nodesNetwork{
				Subnet:           *subnets[0].SubnetId,
				AvailabilityZone: *subnets[0].AvailabilityZone,
				NodeID:           i,
			})
		}
		return dist
	}

	total := 0
	for i := 0; i < len(subnets); i++ {
		dist = append(dist, nodesNetwork{
			Subnet:           *subnets[i].SubnetId,
			AvailabilityZone: *subnets[i].AvailabilityZone,
			NodeID:           i,
		})
		total++
	}
	if total < 5 {
		for i := 1; i < 6-total; i++ {
			n := dist[i]
			n.NodeID = total
			dist = append(dist, n)
			total++
		}
	}
	if total%2 == 0 {
		// Append an additional item to make sure we have an odd number of nodes in total.
		n := dist[0]
		n.NodeID = total
		dist = append(dist, n)
	}
	return dist
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
	nodesPerSubnet, err := c.calcNodesPerSubnet(p.Networks)
	if err != nil {
		return err
	}

	stackName := makeMasterPoolStackName(p.ClusterName, "")
	templateBody, err := renderMasterStackTemplate(p, amiID, elbName, assetsBucketName, nodesPerSubnet, kubeAPIURL, stackName)
	if err != nil {
		return err
	}

	// To ensure stack resources inherit cluster-name.
	tags := make(map[string]string)
	tags[clusterNameTagKey] = p.ClusterName
	tags[stackTypeTagKey] = masterPoolStackType

	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(templateBody),
		Tags:         makeStackTags(tags),
		Capabilities: aws.StringSlice([]string{
			cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
	}
	return c.createStack(stack)
}

func (c *Cloud) calcNodesPerSubnet(networks []string) (map[string]int, error) {
	subnets, err := c.describeSubnets(networks)
	m := make(map[string]int)
	if err != nil {
		return m, err
	}

	dist := getNodesDistributionAcrossNetworks(subnets)
	for _, n := range dist {
		m[n.Subnet]++
	}
	return m, nil
}

// makeMasterPoolStackName returns master stack name for either blue or green stack.
func makeMasterPoolStackName(clusterName, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-%s-%s", clusterName, masterPoolStackType, part)
}

func (c *Cloud) createELBStack(cluster model.Cluster, vpcID, infraStackName string) error {
	templateBody, err := renderELBStackTemplate(cluster, vpcID)
	if err != nil {
		return err
	}

	// To ensure stack resources inherit cluster-name.
	tags := make(map[string]string)
	tags[clusterNameTagKey] = cluster.Name
	tags[stackTypeTagKey] = elbStackType

	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(makeELBStackName(cluster.Name)),
		Tags:         makeStackTags(tags),
		TemplateBody: aws.String(templateBody),
	}
	return c.createStack(stack)
}

// makeELBStackName returns ELB stack name.
// There is no blue/green updates for ELB stack. Updates are handled in place.
func makeELBStackName(clusterName string) string {
	return fmt.Sprintf("keto-%s-%s", clusterName, elbStackType)
}

func (c *Cloud) createComputePoolStack(p model.ComputePool, infraStackName string, amiID string, kubeAPIURL string) error {
	stackName := makeComputePoolStackName(p.ClusterName, p.Name, "")
	templateBody, err := renderComputeStackTemplate(p, amiID, kubeAPIURL, stackName)
	if err != nil {
		return err
	}

	// To ensure stack resources inherit cluster-name.
	tags := make(map[string]string)
	tags[clusterNameTagKey] = p.ClusterName
	tags[stackTypeTagKey] = computePoolStackType

	stack := &cloudformation.CreateStackInput{
		StackName:    aws.String(stackName),
		TemplateBody: aws.String(templateBody),
		Tags:         makeStackTags(tags),
		Capabilities: aws.StringSlice([]string{
			cloudformation.CapabilityCapabilityIam, cloudformation.CapabilityCapabilityNamedIam}),
	}
	return c.createStack(stack)
}

// makeComputePoolStackName returns compute pool stack name for either blue or
// green stack.
func makeComputePoolStackName(clusterName, name, part string) string {
	if part == "" {
		part = blueStack
	}
	return fmt.Sprintf("keto-%s-%s-%s", clusterName, name, part)
}

func makeStackTags(m map[string]string) []*cloudformation.Tag {
	tags := []*cloudformation.Tag{}
	if m != nil {
		for k, v := range m {
			tags = append(tags, &cloudformation.Tag{
				Key:   aws.String(k),
				Value: aws.String(v),
			})
		}
	}

	// Add tags that apply to all stacks.
	tags = append(tags, &cloudformation.Tag{
		Key:   aws.String(managedByKetoTagKey),
		Value: aws.String(managedByKetoTagValue),
	})

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
	if resp.StackId == nil {
		return fmt.Errorf("failed to create %q stack, stack id is nil in response", *in.StackName)
	}

	return c.waitForStackOperationCompletion(*resp.StackId)
}

func (c *Cloud) validateStackTemplate(tpl *string) error {
	params := &cloudformation.ValidateTemplateInput{
		TemplateBody: tpl,
	}
	_, err := c.cf.ValidateTemplate(params)
	return err
}

func (c Cloud) deleteStack(name string) error {
	params := &cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	}
	if _, err := c.cf.DeleteStack(params); err != nil {
		return err
	}

	return c.waitForStackOperationCompletion(name)
}

// waitForStackOperationCompletion returns an error if a stack
// create/update/delete operation fails. Rollback status also returns an error
// to indicate a failure. Otherwise an error returned is nil.
func (c *Cloud) waitForStackOperationCompletion(id string) error {
	for {
		s, err := c.getStack(id)
		if s.StackId == nil {
			return nil
		}
		switch {
		case err != nil:
			return err
		// wait for any status that is in progress to complete
		case strings.HasSuffix(*s.StackStatus, stackStatusInProgressSuffix):
			// do nothing
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
