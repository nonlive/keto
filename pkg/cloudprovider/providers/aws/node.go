package aws

import (
	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Node returns an implementation of Node interface for AWS Cloud.
func (c *Cloud) Node() (cloudprovider.Node, bool) {
	return c, true
}

// GetNodeData returns model.NodeData which contains information like node
// labels, kube version, etc.
func (c Cloud) GetNodeData() (model.NodeData, error) {
	metadata, err := getInstanceMetadata()
	if err != nil {
		return model.NodeData{}, err
	}

	tags, err := c.getInstanceTags(metadata.InstanceID)
	if err != nil {
		return model.NodeData{}, err
	}

	data := model.NodeData{}
	labels := model.Labels{}

	for _, t := range tags {
		if *t.Key == kubeAPIURLTagKey {
			data.KubeAPIURL = *t.Value
		}

		if *t.Key == clusterNameTagKey {
			data.ClusterName = *t.Value
		}

		if *t.Key == kubeVersionTagKey {
			data.KubeVersion = *t.Value
		}

		// Skip over reserved tags.
		if tagReserved(*t.Key) {
			continue
		}
		labels[*t.Key] = *t.Value
	}
	data.Labels = labels

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

func (c Cloud) getInstanceTags(id string) ([]*ec2.TagDescription, error) {
	params := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(id),
				},
			},
		},
	}

	resp, err := c.ec2.DescribeTags(params)
	if err != nil {
		return resp.Tags, err
	}

	return resp.Tags, nil
}

// GetAssets gets assets from a cloud.
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
