package aws

import (
	"fmt"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/keto/util"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

// Node returns an implementation of Node interface for AWS Cloud.
func (c *Cloud) Node() (cloudprovider.Node, bool) {
	return c, true
}

// GetNodeData returns model.NodeData which contains information like node
// labels, kube version, etc.
func (c Cloud) GetNodeData() (model.NodeData, error) {
	var data model.NodeData

	outputs, err := c.getNodeStackOutputs()
	if err != nil {
		return data, err
	}

	// Extract fields from stack outputs.
	for _, o := range outputs {
		if *o.OutputKey == kubeAPIURLOutputKey {
			data.KubeAPIURL = *o.OutputValue
		}
		if *o.OutputKey == clusterNameOutputKey {
			data.ClusterName = *o.OutputValue
		}
		if *o.OutputKey == kubeVersionOutputKey {
			data.KubeVersion = *o.OutputValue
		}
		if *o.OutputKey == labelsOutputKey {
			data.Labels = util.KVsToLabels(strings.Split(*o.OutputValue, "="))
		}
	}

	return data, nil
}

// getInstanceMetadata returns an instance metadata document from EC2 metadata service.
func getInstanceMetadata() (ec2metadata.EC2InstanceIdentityDocument, error) {
	sess, err := session.NewSession(&aws.Config{})
	if err != nil {
		return ec2metadata.EC2InstanceIdentityDocument{}, err
	}
	metadataService := ec2metadata.New(sess)
	return metadataService.GetInstanceIdentityDocument()
}

// GetAssets gets assets from a cloud.
func (c *Cloud) GetAssets() (model.Assets, error) {
	var a model.Assets

	outputs, err := c.getNodeStackOutputs()
	if err != nil {
		return a, err
	}

	var bucket string
	for _, o := range outputs {
		if *o.OutputKey == assetsBucketNameOutputKey {
			bucket = *o.OutputValue
			break
		}
	}

	etcdCACert, err := c.getS3Object(bucket, etcdCACertObjectName)
	if err != nil {
		return a, err
	}
	etcdCAKey, err := c.getS3Object(bucket, etcdCAKeyObjectName)
	if err != nil {
		return a, err
	}
	kubeCACert, err := c.getS3Object(bucket, kubeCACertObjectName)
	if err != nil {
		return a, err
	}
	kubeCAKey, err := c.getS3Object(bucket, kubeCAKeyObjectName)
	if err != nil {
		return a, err
	}

	a.EtcdCAKey = etcdCAKey
	a.EtcdCACert = etcdCACert
	a.KubeCAKey = kubeCAKey
	a.KubeCACert = kubeCACert

	return a, nil
}

// Returns cloudformation stack outputs. It is the stack that the node was
// created by. Should only be used from a node.
func (c Cloud) getNodeStackOutputs() ([]*cloudformation.Output, error) {
	var outputs []*cloudformation.Output

	metadata, err := getInstanceMetadata()
	if err != nil {
		return outputs, err
	}

	stackName, err := c.getResourceTagValue(metadata.InstanceID, "aws:cloudformation:stack-name")
	if err != nil {
		return outputs, fmt.Errorf("aws:cloudformation:stack-name tag not found")
	}

	stack, err := c.getStack(stackName)
	if err != nil {
		return outputs, fmt.Errorf("failed to describe %q stack: %v", stackName, err)
	}

	return stack.Outputs, nil
}
