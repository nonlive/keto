package aws

import (
	"io/ioutil"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Node returns an implementation of Node interface for AWS Cloud.
func (c *Cloud) Node() (cloudprovider.Node, bool) {
	return c, true
}

// GetKubeAPIURL returns a full URL to Kubernetes API.
func (c *Cloud) GetKubeAPIURL() (string, error) {
	instanceID, err := c.getInstanceID()
	if err != nil {
		return "", err
	}
	return c.getResourceTagValue(instanceID, kubeAPIURLTagKey)
}

// GetKubeVersion returns a kubernetes version string.
func (c *Cloud) GetKubeVersion() (string, error) {
	instanceID, err := c.getInstanceID()
	if err != nil {
		return "", err
	}
	return c.getResourceTagValue(instanceID, kubeVersionTagKey)
}

// getInstanceID returns an instance ID from EC2 metadata service.
func (c Cloud) getInstanceID() (string, error) {
	return c.ec2Metadata.GetMetadata("instance-id")
}

// GetAssets gets assets onto a filesystem.
func (c *Cloud) GetAssets() (model.Assets, error) {
	a := model.Assets{}

	instanceID, err := c.getInstanceID()
	if err != nil {
		return a, err
	}
	bucket, err := c.getResourceTagValue(instanceID, assetsBucketNameTagKey)
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

func (c Cloud) getS3Object(bucket, objectName string) ([]byte, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectName),
	}

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
