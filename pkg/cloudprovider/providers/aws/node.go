package aws

import (
	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Node returns an implementation of Node interface for AWS Cloud.
func (c *Cloud) Node() (cloudprovider.Node, bool) {
	return c, true
}

// GetKubeAPIURL returns a full URL to Kubernetes API.
func (c *Cloud) GetKubeAPIURL() (string, error) {
	metadata, err := getInstanceMetadata()
	if err != nil {
		return "", err
	}
	return c.getResourceTagValue(metadata.InstanceID, kubeAPIURLTagKey)
}

// GetClusterName returns the cluster-name value.
func (c *Cloud) GetClusterName() (string, error) {
	metadata, err := getInstanceMetadata()
	if err != nil {
		return "", err
	}
	return c.getResourceTagValue(metadata.InstanceID, clusterNameTagKey)
}

// GetKubeVersion returns a kubernetes version string.
func (c *Cloud) GetKubeVersion() (string, error) {
	metadata, err := getInstanceMetadata()
	if err != nil {
		return "", err
	}
	return c.getResourceTagValue(metadata.InstanceID, kubeVersionTagKey)
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

// GetAssets gets assets onto a filesystem.
func (c *Cloud) GetAssets() (model.Assets, error) {
	a := model.Assets{}

	metadata, err := getInstanceMetadata()
	if err != nil {
		return a, err
	}
	bucket, err := c.getResourceTagValue(metadata.InstanceID, assetsBucketNameTagKey)
	if err != nil {
		return a, err
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
