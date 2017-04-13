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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
)

const (
	// ProviderName is the name of this provider.
	ProviderName = "aws"

	// CoreOS AWS account ID used for AMI id lookup
	coreOSAWSAccountID = "595879546273"

	// managedByKeto tag is needed for the cloudprovider to know which cloud
	// resources are managed by keto.
	managedByKetoTagKey   = "managed-by-keto"
	managedByKetoTagValue = "true"
	clusterNameTagKey     = "cluster-name"
)

var (
	// ErrNotImplemented defines an error for not implemented features.
	ErrNotImplemented = errors.New("not implemented")
)

// Cloud is an implementation of cloudprovider.Interface.
type Cloud struct {
	cf          *cloudformation.CloudFormation
	ec2         *ec2.EC2
	elb         *elb.ELB
	ec2Metadata *ec2metadata.EC2Metadata
}

// Compile-time check whether Cloud type value implements
// cloudprovider.Interface interface.
var _ cloudprovider.Interface = (*Cloud)(nil)

// ProviderName returns the cloud provider ID.
func (c *Cloud) ProviderName() string {
	return ProviderName
}

// Clusters returns an implementation of Clusters interface for AWS Cloud.
func (c *Cloud) Clusters() (cloudprovider.Clusters, bool) {
	return c, true
}

// CreateCluster creates a new cluster, by creating ENIs, volumes and other
// cluster infra related resources.
// TODO(vaijab): should rename this to CreateClusterInfra() instead?
func (c *Cloud) CreateCluster(cluster model.Cluster) error {
	infraStackExists, err := c.stackExists(makeClusterInfraStackName(cluster.Name))
	if err != nil {
		return err
	}
	if infraStackExists {
		return errors.New("cluster already exists")
	}

	subnets, err := c.describeSubnets(cluster.MasterPool.Networks)
	if err != nil {
		return err
	}
	vpcID, err := getVpcIDFromSubnetList(subnets)
	if err != nil {
		return err
	}

	if err := c.createClusterInfraStack(cluster.Name, vpcID, subnets); err != nil {
		return err
	}
	return nil
}

// GetClusters returns a cluster by name or all clusters in the region.
func (c *Cloud) GetClusters(name string) ([]*model.Cluster, error) {
	clusters := []*model.Cluster{}

	stacks, err := c.getStacksByType(clusterInfraStackType)
	if err != nil {
		return clusters, err
	}

outer:
	for _, s := range stacks {
		c := &model.Cluster{}
		for _, tag := range s.Tags {
			if *tag.Key == clusterNameTagKey && *tag.Value != "" {
				// if filtered by cluster name and the stack tag does not
				// match it, skip over the stack
				if name != "" && *tag.Value != name {
					continue outer
				}
				c.Name = *tag.Value
			}
		}

		c.Labels = getStackLabels(s)
		clusters = append(clusters, c)
	}
	return clusters, nil
}

// DescribeCluster describes a given cluster.
func (c *Cloud) DescribeCluster(name string) error {
	return ErrNotImplemented
}

// getKubeAPIURLFromELB returns a full Kubernetes API URL from an ELB.
func (c Cloud) getKubeAPIURLFromELB(clusterName string) (string, error) {
	elbName, err := c.getELBName(clusterName)
	if err != nil {
		return "", err
	}
	params := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			aws.String(elbName),
		},
	}
	resp, err := c.elb.DescribeLoadBalancers(params)
	if err != nil {
		return "", err
	}
	if len(resp.LoadBalancerDescriptions) == 0 {
		// TODO(vaijab) use awserr to access the underlying error
		return "", errors.New("no load balancers found")
	}
	return formatKubeAPIURL(*resp.LoadBalancerDescriptions[0].DNSName), nil
}

func formatKubeAPIURL(host string) string {
	// For some reason kubernetes does not like mixed-case dns names.
	return "https://" + strings.ToLower(host)
}

// getELBName returns an ELB name from the ELB stack for a given given cluster.
func (c Cloud) getELBName(clusterName string) (string, error) {
	res, err := c.getStackResources(makeELBStackName(clusterName))
	if err != nil {
		return "", err
	}
	for _, r := range res {
		if *r.ResourceType == "AWS::ElasticLoadBalancing::LoadBalancer" {
			return *r.PhysicalResourceId, nil
		}
	}
	return "", nil
}

// DeleteCluster deletes a cluster.
func (c *Cloud) DeleteCluster(name string) error {
	return ErrNotImplemented
}

// GetMasterPersistentIPs returns a map of master persistent NodeID
// values and private IPs for a given clusterName.
func (c Cloud) GetMasterPersistentIPs(clusterName string) (map[string]string, error) {
	m := make(map[string]string)

	enis, err := c.describePersistentENIs(clusterName)
	if err != nil {
		return m, err
	}

	for _, n := range enis {
		if id := getENINodeID(n); id != "" {
			m[id] = *n.PrivateIpAddress
		}
	}
	return m, nil
}

// getENINodeID extract a NodeID tag value from an ENI. Return an empty string
// if no such tag exists.
func getENINodeID(n *ec2.NetworkInterface) string {
	if n == nil {
		return ""
	}
	for _, tag := range n.TagSet {
		if *tag.Key == "NodeID" && *tag.Value != "" {
			return *tag.Value
		}
	}
	return ""
}

// describePersistentENIs returns a list of persistent master network
// interfaces, that are used by etcd.
func (c Cloud) describePersistentENIs(clusterName string) ([]*ec2.NetworkInterface, error) {
	params := &ec2.DescribeNetworkInterfacesInput{
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
	}
	resp, err := c.ec2.DescribeNetworkInterfaces(params)
	if err != nil {
		return []*ec2.NetworkInterface{}, err
	}
	return resp.NetworkInterfaces, nil
}

// NodePooler returns an implementation of NodePooler interface for
// AWS Cloud.
func (c *Cloud) NodePooler() (cloudprovider.NodePooler, bool) {
	return c, true
}

// CreateMasterPool creates a master node pool.
func (c *Cloud) CreateMasterPool(p model.MasterPool) error {
	// At this point a cluster infra has created persistent ENIs, so master
	// nodes should be created in the same subnets as ENIs, we just
	// overwrite MasterPool.Networks.
	enis, err := c.describePersistentENIs(p.ClusterName)
	if err != nil {
		return err
	}
	p.Networks = []string{}
	for _, n := range enis {
		p.Networks = append(p.Networks, *n.SubnetId)
	}

	elbStackExists, err := c.stackExists(makeELBStackName(p.ClusterName))
	if err != nil {
		return err
	}
	if !elbStackExists {
		if err := c.createLoadBalancer(p); err != nil {
			return err
		}
	}

	// TODO(vaijab) should be passed in through CLI, but need to figure out
	// some sort of validation and CoreOS version to AMI name mapping.
	p.OSVersion = "CoreOS-beta-1325.2.0-hvm"
	amiID, err := c.getAMIByName(p.OSVersion)
	if err != nil {
		return err
	}

	elbName, err := c.getELBName(p.ClusterName)
	if err != nil {
		return err
	}
	kubeAPIURL, err := c.getKubeAPIURLFromELB(p.ClusterName)
	if err != nil {
		return err
	}
	infraStackName := makeClusterInfraStackName(p.ClusterName)
	if err := c.createMasterPoolStack(p, infraStackName, amiID, elbName, kubeAPIURL); err != nil {
		return err
	}
	return nil
}

// createLoadBalancer ensures a load balancer is created.
func (c *Cloud) createLoadBalancer(p model.MasterPool) error {
	subnets, err := c.describeSubnets(p.Networks)
	if err != nil {
		return err
	}
	vpcID, err := getVpcIDFromSubnetList(subnets)
	if err != nil {
		return err
	}

	infraStackName := makeClusterInfraStackName(p.ClusterName)
	if err := c.createELBStack(p, vpcID, infraStackName); err != nil {
		return err
	}
	return nil
}

// CreateComputePool creates a compute node pool.
func (c *Cloud) CreateComputePool(nodePool model.ComputePool) error {
	return ErrNotImplemented
}

// GetMasterPools returns a list of master pools. Pools can be filtered by
// their name / cluster.
// TODO(vaijab): refactor below into a shared function to get nodepools?
func (c *Cloud) GetMasterPools(clusterName, name string) ([]*model.MasterPool, error) {
	pools := []*model.MasterPool{}

	stacks, err := c.getStacksByType(masterPoolStackType)
	if err != nil {
		return pools, err
	}

outer:
	for _, s := range stacks {
		p := &model.MasterPool{}
		for _, tag := range s.Tags {
			if *tag.Key == clusterNameTagKey && *tag.Value != "" {
				if clusterName != "" && *tag.Value != clusterName {
					continue outer
				}
				p.ClusterName = *tag.Value
			}
			if *tag.Key == poolNameTagKey && *tag.Value != "" {
				if name != "" && *tag.Value != name {
					continue outer
				}
				p.Name = *tag.Value
			}
			if *tag.Key == kubeVersionTagKey {
				p.KubeVersion = *tag.Value
			}
			if *tag.Key == osVersionTagKey {
				p.OSVersion = *tag.Value
			}
			if *tag.Key == machineTypeTagKey {
				p.MachineType = *tag.Value
			}
			if *tag.Key == diskSizeTagKey {
				i, err := strconv.Atoi(*tag.Value)
				if err != nil {
					return pools, err
				}
				p.DiskSize = i
			}
		}

		p.Labels = getStackLabels(s)
		pools = append(pools, p)
	}
	return pools, nil
}

// GetComputePools returns a list of compute pools. Pools can be filtered by
// their name / cluster.
// TODO(vaijab): refactor below into a shared function to get nodepools?
func (c *Cloud) GetComputePools(clusterName, name string) ([]*model.ComputePool, error) {
	pools := []*model.ComputePool{}

	stacks, err := c.getStacksByType(masterPoolStackType)
	if err != nil {
		return pools, err
	}

outer:
	for _, s := range stacks {
		p := &model.ComputePool{}
		for _, tag := range s.Tags {
			if *tag.Key == clusterNameTagKey && *tag.Value != "" {
				if clusterName != "" && *tag.Value != clusterName {
					continue outer
				}
				p.ClusterName = *tag.Value
			}
			if *tag.Key == poolNameTagKey && *tag.Value != "" {
				if name != "" && *tag.Value != name {
					continue outer
				}
				p.Name = *tag.Value
			}
			if *tag.Key == kubeVersionTagKey {
				p.KubeVersion = *tag.Value
			}
			if *tag.Key == osVersionTagKey {
				p.OSVersion = *tag.Value
			}
			if *tag.Key == machineTypeTagKey {
				p.MachineType = *tag.Value
			}
			if *tag.Key == diskSizeTagKey {
				i, err := strconv.Atoi(*tag.Value)
				if err != nil {
					return pools, err
				}
				p.DiskSize = i
			}
		}

		p.Labels = getStackLabels(s)
		pools = append(pools, p)
	}
	return pools, nil
}

// DescribeNodePool lists nodes pools.
func (c *Cloud) DescribeNodePool() error {
	return ErrNotImplemented
}

// UpgradeNodePool upgrades a node pool.
func (c *Cloud) UpgradeNodePool() error {
	return ErrNotImplemented
}

// DeleteNodePool deletes a node pool.
func (c *Cloud) DeleteNodePool(clusterName, name string) error {
	return ErrNotImplemented
}

// getVpcIDFromSubnetList checks given subnets belong to the same VPC and
// returns the VPC ID, if not, an empty string is returned with an error.
func getVpcIDFromSubnetList(subnets []*ec2.Subnet) (string, error) {
	if !subnetsBelongToSameVPC(subnets) {
		return "", errors.New("subnets do not belong to the same VPC")
	}
	return *subnets[0].VpcId, nil
}

// subnetsBelongToSameVPC returns true if given subnets belong to the same VPC.
func subnetsBelongToSameVPC(subnets []*ec2.Subnet) bool {
	m := make(map[string]bool)
	for _, v := range subnets {
		m[*v.VpcId] = true
	}
	return len(m) == 1
}

// describeSubnets returns a slice of subnet structs as well as an error value.
func (c *Cloud) describeSubnets(subnetIDs []string) ([]*ec2.Subnet, error) {
	// AWS expects pointers instead of string values, got to convert each value.
	sp := []*string{}
	for _, s := range subnetIDs {
		sp = append(sp, aws.String(s))
	}

	subnets := []*ec2.Subnet{}
	resp, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{SubnetIds: sp})
	if err != nil {
		return subnets, err
	}
	for _, s := range resp.Subnets {
		subnets = append(subnets, s)
	}
	return subnets, nil
}

// getAMIByName returns AMI ID for a given AMI name.
func (c *Cloud) getAMIByName(name string) (string, error) {
	params := &ec2.DescribeImagesInput{
		Owners: []*string{aws.String(coreOSAWSAccountID)},
		Filters: []*ec2.Filter{
			{Name: aws.String("name"), Values: []*string{aws.String(name)}},
			{Name: aws.String("virtualization-type"), Values: []*string{aws.String("hvm")}},
			{Name: aws.String("state"), Values: []*string{aws.String("available")}},
		},
	}
	resp, err := c.ec2.DescribeImages(params)
	if err != nil {
		return "", err
	}
	if len(resp.Images) > 0 {
		return *resp.Images[0].ImageId, nil
	}
	return "", fmt.Errorf("image %q not found", name)
}

// getResourceTagValue returns a value of the tag key of the resourceID.
func (c Cloud) getResourceTagValue(resourceID, key string) (string, error) {
	params := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(resourceID),
				},
			},
			{
				Name: aws.String("key"),
				Values: []*string{
					aws.String(key),
				},
			},
		},
	}

	resp, err := c.ec2.DescribeTags(params)
	if err != nil {
		return "", err
	}
	for _, t := range resp.Tags {
		if *t.Key == key {
			return *t.Value, nil
		}
	}
	return "", nil
}

// init registers AWS cloud with the cloudprovider.
func init() {
	// f knows how to initialize the cloud with given config
	f := func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	}
	cloudprovider.Register(ProviderName, f)
}

// newCloud creates a new instance of AWS Cloud. It takes an optional io.Reader
// argument as a cloud config.
func newCloud(config io.Reader) (*Cloud, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState:       session.SharedConfigEnable,
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
	}))

	// If region has not been provided, let's try to get it from an EC2
	// metadata service and fail if we cannot get that way.
	if *sess.Config.Region == "" {
		s := session.Must(session.NewSession(aws.NewConfig().WithMaxRetries(0)))
		m := ec2metadata.New(s)
		r, err := m.Region()
		if err != nil {
			return &Cloud{}, errors.New("could not find region configuration")
		}
		sess.Config.Region = &r
	}

	c := &Cloud{
		cf:          cloudformation.New(sess),
		ec2:         ec2.New(sess),
		elb:         elb.New(sess),
		ec2Metadata: ec2metadata.New(sess),
	}

	return c, nil
}
