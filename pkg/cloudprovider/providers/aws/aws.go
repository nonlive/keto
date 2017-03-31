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
	"os"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
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
	cf     *cloudformation.CloudFormation
	ec2    *ec2.EC2
	elb    *elb.ELB
	region string
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

// ListClusters returns a cluster by name or all clusters in the region.
func (c *Cloud) ListClusters(name string) ([]*model.Cluster, error) {
	return []*model.Cluster{}, nil
}

// DescribeCluster describes a given cluster.
func (c *Cloud) DescribeCluster(name string) error {
	return ErrNotImplemented
}

// GetKubeAPIURL returns a ful Kubernetes API URL.
func (c Cloud) GetKubeAPIURL(clusterName string) (string, error) {
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
		return "", errors.New("no load balancers found")
	}
	// For some reason kubernetes does not like mixed-case dns names.
	return "https://" + strings.ToLower(*resp.LoadBalancerDescriptions[0].DNSName), nil
}

// getELBName returns an ELB name from the ELB stack for a given given cluster.
func (c Cloud) getELBName(clusterName string) (string, error) {
	stackName := makeELBStackName(clusterName)
	s, err := c.getStack(stackName)
	if err != nil {
		return "", err
	}
	for _, o := range s.Outputs {
		if *o.OutputKey == "ELB" {
			return *o.OutputValue, nil
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
	// At this point a cluster has been created with persistent ENIs, master
	// nodes should be created in the same subnets as ENIs, so we just
	// overwrite MasterPool Networks.
	enis, err := c.describePersistentENIs(p.ClusterName)
	if err != nil {
		return err
	}
	p.Networks = []string{}
	for _, n := range enis {
		p.Networks = append(p.Networks, *n.SubnetId)
	}

	infraStackName := makeClusterInfraStackName(p.ClusterName)
	// TODO(vaijab) should be passed in through CLI, but need to figure out
	// some sort of validation and CoreOS version to AMI name mapping.
	amiID, err := c.getAMIByName("CoreOS-beta-1325.2.0-hvm")
	if err != nil {
		return err
	}
	// TODO(vaijab) keto should create a key pair.
	sshKeyPairName := os.Getenv("USER")

	if err := c.createMasterStack(p, infraStackName, amiID, sshKeyPairName); err != nil {
		return err
	}
	return nil
}

// CreateComputePool creates a compute node pool.
func (c *Cloud) CreateComputePool(nodePool model.ComputePool) error {
	return ErrNotImplemented
}

// ListNodePools lists node pools that belong to a given clusterName.
func (c *Cloud) ListNodePools(clusterName string) ([]*model.NodePool, error) {
	var pools []*model.NodePool
	return pools, ErrNotImplemented
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

// LoadBalancer returns an implementation of LoadBalancer interface for
// AWS Cloud.
func (c *Cloud) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return c, true
}

// GetLoadBalancer returns a load balancer given cluster name.
func (c *Cloud) GetLoadBalancer(clusterName string) error {
	return ErrNotImplemented
}

// CreateLoadBalancer ensures a load balancer is created.
func (c *Cloud) CreateLoadBalancer(p model.MasterPool) error {
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

// UpdateLoadBalancer updates a load balancer.
func (c *Cloud) UpdateLoadBalancer() error {
	return ErrNotImplemented
}

// DeleteLoadBalancer ensures a load balancer is deleted.
func (c *Cloud) DeleteLoadBalancer(clusterName string) error {
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

	c := &Cloud{
		cf:     cloudformation.New(sess),
		ec2:    ec2.New(sess),
		elb:    elb.New(sess),
		region: *sess.Config.Region,
	}
	return c, nil
}
