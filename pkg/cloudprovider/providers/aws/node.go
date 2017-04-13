package aws

import (
	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
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
