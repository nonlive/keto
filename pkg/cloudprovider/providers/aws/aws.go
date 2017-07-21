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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
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

	clusterNameTagKey = "cluster-name"
	stackTypeTagKey   = "stack-type"

	etcdCACertObjectName = "etcd_ca.crt"
	etcdCAKeyObjectName  = "etcd_ca.key"
	kubeCACertObjectName = "kube_ca.crt"
	kubeCAKeyObjectName  = "kube_ca.key"
)

var (
	// ErrNotImplemented defines an error for not implemented features.
	ErrNotImplemented = errors.New("not implemented")
)

// Cloud is an implementation of cloudprovider.Interface.
type Cloud struct {
	Logger cloudprovider.Logger
	cf     cloudformationiface.CloudFormationAPI
	ec2    ec2iface.EC2API
	elb    elbiface.ELBAPI
	s3     s3iface.S3API
	r53    route53iface.Route53API
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

// CreateClusterInfra creates a new cluster, by creating ENIs, volumes and other
// cluster infra related resources.
func (c *Cloud) CreateClusterInfra(cluster model.Cluster) error {
	// Check whether hosted zone exists before creating any stacks.
	if cluster.DNSZone != "" {
		params := &route53.ListHostedZonesByNameInput{
			DNSName: aws.String(cluster.DNSZone),
		}
		r, err := c.r53.ListHostedZonesByName(params)
		if err != nil {
			return fmt.Errorf("failed to list route53 dns zone: %v", err)
		}
		if r.HostedZones == nil {
			return fmt.Errorf("dns zone %q does not exist", cluster.DNSZone)
		}
	}

	subnets, err := c.describeSubnets(cluster.MasterPool.Networks)
	if err != nil {
		return err
	}
	c.Logger.Printf("getting VPC ID from a list of subnets %v", cluster.MasterPool.Networks)
	vpcID, err := getVpcIDFromSubnetList(subnets)
	if err != nil {
		return err
	}
	c.Logger.Printf("found %q VPC ID", vpcID)

	if err := c.createClusterInfraStack(cluster, vpcID, subnets); err != nil {
		return err
	}

	// ELB scheme is determined via Masterpool.Internal
	cluster.MasterPool.Internal = cluster.Internal

	return c.createLoadBalancer(cluster)
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
		for _, o := range s.Outputs {
			if *o.OutputKey == clusterNameOutputKey && *o.OutputValue != "" {
				// if filtered by cluster name and the stack output key does not
				// match it, skip over the stack
				if name != "" && *o.OutputValue != name {
					continue outer
				}
				c.Name = *o.OutputValue
			}
		}

		c.Internal = clusterInternal(s.Outputs)
		c.Labels = getStackLabels(s)
		clusters = append(clusters, c)
	}
	return clusters, nil
}

// clusterInternal checks whether a given list of stack Outputs contains a
// internalClusterOutputKey and returns its value as a bool.
func clusterInternal(outputs []*cloudformation.Output) bool {
	for _, o := range outputs {
		if *o.OutputKey == internalClusterOutputKey && o.OutputValue != nil {
			internal, err := strconv.ParseBool(*o.OutputValue)
			if err != nil {
				return false
			}
			return internal
		}
	}
	return false
}

// DescribeCluster describes a given cluster.
func (c *Cloud) DescribeCluster(name string) error {
	return ErrNotImplemented
}

// getKubeAPIURL returns a full Kubernetes API URL from an ELB stack.
func (c Cloud) getKubeAPIURL(clusterName string) (string, error) {
	stack, err := c.getStack(makeELBStackName(clusterName))
	if err != nil {
		return "", err
	}

	for _, o := range stack.Outputs {
		if *o.OutputKey == "ELBDNS" {
			return formatKubeAPIURL(*o.OutputValue), nil
		}
	}

	return "", err
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
	c.Logger.Printf("deleting compute pools that belong to cluster %q", name)
	if err := c.DeleteComputePool(name, ""); err != nil {
		return err
	}

	c.Logger.Printf("deleting master pool that belongs to cluster %q", name)
	if err := c.DeleteMasterPool(name); err != nil {
		return err
	}

	c.Logger.Printf("deleting ELB stack that belongs to cluster %q", name)
	if err := c.deleteStack(makeELBStackName(name)); err != nil {
		return err
	}

	assets := []string{
		etcdCACertObjectName,
		etcdCAKeyObjectName,
		kubeCACertObjectName,
		kubeCAKeyObjectName,
	}

	bucketName, err := c.getAssetsBucketName(name)
	if err != nil {
		return err
	}
	if err := c.deleteS3Objects(bucketName, assets); err != nil {
		return err
	}

	if err := c.deleteStack(makeClusterInfraStackName(name)); err != nil {
		return err
	}
	return nil
}

func (c Cloud) deleteS3Objects(b string, keys []string) error {
	objects := []*s3.ObjectIdentifier{}
	for _, k := range keys {
		objects = append(objects, &s3.ObjectIdentifier{Key: aws.String(k)})
	}

	params := &s3.DeleteObjectsInput{
		Bucket: aws.String(b),
		Delete: &s3.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}

	c.Logger.Printf("deleting objects %v from S3 bucket %q", keys, b)
	_, err := c.s3.DeleteObjects(params)
	return err
}

func (c Cloud) getS3Object(bucket, objectName string) ([]byte, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	c.Logger.Printf("fetching object %q from S3 bucket %q", objectName, bucket)
	resp, err := c.s3.GetObject(params)
	if err != nil {
		return []byte{}, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
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

// PushAssets pushes assets to an S3 bucket.
func (c *Cloud) PushAssets(clusterName string, a model.Assets) error {
	bucket, err := c.getAssetsBucketName(clusterName)
	if err != nil {
		return err
	}

	// We only need the assets for the initial bootstrap.
	if err := c.putS3Object(bucket, etcdCACertObjectName, a.EtcdCACert); err != nil {
		return err
	}
	if err := c.putS3Object(bucket, etcdCAKeyObjectName, a.EtcdCAKey); err != nil {
		return err
	}
	if err := c.putS3Object(bucket, kubeCACertObjectName, a.KubeCACert); err != nil {
		return err
	}
	if err := c.putS3Object(bucket, kubeCAKeyObjectName, a.KubeCAKey); err != nil {
		return err
	}

	return nil
}

// getAssetsBucketName returns assets S3 bucket name from a cluster infra stack.
func (c Cloud) getAssetsBucketName(clusterName string) (string, error) {
	res, err := c.getStackResources(makeClusterInfraStackName(clusterName))
	if err != nil {
		return "", err
	}
	for _, r := range res {
		if *r.ResourceType == "AWS::S3::Bucket" {
			return *r.PhysicalResourceId, nil
		}
	}
	return "", nil
}

// putS3Object uploads b object as objectName to S3 bucket b. Optionally, an
// expiry time can be set as well.
func (c Cloud) putS3Object(bucket string, objectName string, b []byte) error {
	params := &s3.PutObjectInput{
		Body:   bytes.NewReader(b),
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

	_, err := c.s3.PutObject(params)
	return err
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

	amiID, err := c.getAMIByName(p.CoreOSVersion)
	if err != nil {
		return err
	}

	elbName, err := c.getELBName(p.ClusterName)
	if err != nil {
		return err
	}

	kubeAPIURL, err := c.getKubeAPIURL(p.ClusterName)
	if err != nil {
		return err
	}

	bucket, err := c.getAssetsBucketName(p.ClusterName)
	if err != nil {
		return err
	}

	infraStackName := makeClusterInfraStackName(p.ClusterName)
	return c.createMasterPoolStack(p, infraStackName, amiID, elbName, kubeAPIURL, bucket)
}

// createLoadBalancer ensures a load balancer is created.
func (c *Cloud) createLoadBalancer(cluster model.Cluster) error {
	subnets, err := c.describeSubnets(cluster.MasterPool.Networks)
	if err != nil {
		return err
	}
	vpcID, err := getVpcIDFromSubnetList(subnets)
	if err != nil {
		return err
	}

	infraStackName := makeClusterInfraStackName(cluster.Name)
	return c.createELBStack(cluster, vpcID, infraStackName)
}

// CreateComputePool creates a compute node pool.
// Creating compute pools in different VPCs from where masterpool sits is
// not supported. Mainly due to complexities imposed by AWS.
func (c *Cloud) CreateComputePool(p model.ComputePool) error {
	vpcID, err := c.getClusterVpcID(p.ClusterName)
	if err != nil {
		return err
	}
	subnets, err := c.describeSubnets(p.Networks)
	if err != nil {
		return err
	}
	if len(subnets) == 0 {
		return errors.New("no subnets found")
	}
	if !subnetsBelongToSameVPC(subnets) {
		return errors.New("networks must be part of the same VPC")
	}
	if *subnets[0].VpcId != vpcID {
		return fmt.Errorf("networks must belong to %q VPC", vpcID)
	}

	infraStackName := makeClusterInfraStackName(p.ClusterName)

	amiID, err := c.getAMIByName(p.CoreOSVersion)
	if err != nil {
		return err
	}
	kubeAPIURL, err := c.getKubeAPIURL(p.ClusterName)
	if err != nil {
		return err
	}

	return c.createComputePoolStack(p, infraStackName, amiID, kubeAPIURL)
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
		for _, o := range s.Outputs {
			if *o.OutputKey == clusterNameOutputKey && *o.OutputValue != "" {
				if clusterName != "" && *o.OutputValue != clusterName {
					continue outer
				}
				p.ClusterName = *o.OutputValue
			}
			if *o.OutputKey == poolNameOutputKey && *o.OutputValue != "" {
				if name != "" && *o.OutputValue != name {
					continue outer
				}
				p.Name = *o.OutputValue
			}
			if *o.OutputKey == kubeVersionOutputKey {
				p.KubeVersion = *o.OutputValue
			}
			if *o.OutputKey == coreOSVersionOutputKey {
				p.CoreOSVersion = *o.OutputValue
			}
			if *o.OutputKey == machineTypeOutputKey {
				p.MachineType = *o.OutputValue
			}
			if *o.OutputKey == diskSizeOutputKey {
				i, err := strconv.Atoi(*o.OutputValue)
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

	stacks, err := c.getStacksByType(computePoolStackType)
	if err != nil {
		return pools, err
	}

outer:
	for _, s := range stacks {
		p := &model.ComputePool{}
		for _, o := range s.Outputs {
			if *o.OutputKey == clusterNameOutputKey && *o.OutputValue != "" {
				if clusterName != "" && *o.OutputValue != clusterName {
					continue outer
				}
				p.ClusterName = *o.OutputValue
			}
			if *o.OutputKey == poolNameOutputKey && *o.OutputValue != "" {
				if name != "" && *o.OutputValue != name {
					continue outer
				}
				p.Name = *o.OutputValue
			}
			if *o.OutputKey == kubeVersionOutputKey {
				p.KubeVersion = *o.OutputValue
			}
			if *o.OutputKey == coreOSVersionOutputKey {
				p.CoreOSVersion = *o.OutputValue
			}
			if *o.OutputKey == machineTypeOutputKey {
				p.MachineType = *o.OutputValue
			}
			if *o.OutputKey == diskSizeOutputKey {
				i, err := strconv.Atoi(*o.OutputValue)
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

// DeleteMasterPool deletes a master node pool.
func (c *Cloud) DeleteMasterPool(clusterName string) error {
	stacks, err := c.getStacksByType(masterPoolStackType)
	if err != nil {
		return err
	}

	for _, s := range stacks {
		for _, o := range s.Outputs {
			if *o.OutputKey == clusterNameOutputKey && *o.OutputValue == clusterName {
				if err := c.deleteStack(*s.StackId); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// DeleteComputePool deletes a node pool.
func (c *Cloud) DeleteComputePool(clusterName, name string) error {
	stacks, err := c.getStacksByType(computePoolStackType)
	if err != nil {
		return err
	}

	matched := func(outputs []*cloudformation.Output) bool {
		n := 0
		for _, o := range outputs {
			if *o.OutputKey == clusterNameOutputKey && *o.OutputValue == clusterName {
				n++
			}
			if name != "" {
				if *o.OutputKey == poolNameOutputKey && *o.OutputValue == name {
					n++
				}
			}
			if name == "" && *o.OutputKey == poolNameOutputKey {
				n++
			}
		}
		return n == 2
	}

	for _, s := range stacks {
		if matched(s.Outputs) {
			if err := c.deleteStack(*s.StackId); err != nil {
				return err
			}
		}
	}

	return nil
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

// getClusterVpcID returns the cluster vpc id from cluster infra stack.
func (c Cloud) getClusterVpcID(name string) (string, error) {
	s, err := c.getStack(makeClusterInfraStackName(name))
	if err != nil {
		return "", err
	}

	for _, o := range s.Outputs {
		if *o.OutputKey == "VpcID" {
			return *o.OutputValue, nil
		}
	}
	return "", nil
}

// describeSubnets returns a slice of subnet structs as well as an error value.
func (c *Cloud) describeSubnets(subnetIDs []string) ([]*ec2.Subnet, error) {
	// AWS expects pointers instead of string values, got to convert each value.
	sp := []*string{}
	for _, s := range subnetIDs {
		sp = append(sp, aws.String(s))
	}

	c.Logger.Printf("describing a list of subnets %v", subnetIDs)
	subnets := []*ec2.Subnet{}
	resp, err := c.ec2.DescribeSubnets(&ec2.DescribeSubnetsInput{SubnetIds: sp})
	if err != nil {
		return subnets, err
	}

	subnets = append(subnets, resp.Subnets...)
	c.Logger.Printf("received subnets description: %+v", subnets)

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

	c.Logger.Printf("getting tag %q value of resource %q", key, resourceID)
	resp, err := c.ec2.DescribeTags(params)
	if err != nil {
		return "", err
	}
	for _, t := range resp.Tags {
		if *t.Key == key {
			c.Logger.Printf("got %q as value of the tag %q", *t.Value, *t.Key)
			return *t.Value, nil
		}
	}
	return "", nil
}

// init registers AWS cloud with the cloudprovider.
func init() {
	// f knows how to initialize the cloud
	f := func(l cloudprovider.Logger) (cloudprovider.Interface, error) {
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
				return &Cloud{}, errors.New("unable to determine region")
			}
			sess.Config.Region = &r
		}

		return newCloud(sess, l)
	}
	cloudprovider.Register(ProviderName, f)
}

// newCloud creates a new instance of AWS Cloud given sess session.
func newCloud(sess *session.Session, l cloudprovider.Logger) (*Cloud, error) {
	c := &Cloud{
		Logger: l,
		cf:     cloudformation.New(sess),
		ec2:    ec2.New(sess),
		elb:    elb.New(sess),
		s3:     s3.New(sess),
		r53:    route53.New(sess),
	}
	return c, nil
}
