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

package controller

import (
	"errors"
	"fmt"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/UKHomeOffice/keto/pkg/userdata"
)

var (
	// ErrNotImplemented is an error for not implemented features.
	ErrNotImplemented = errors.New("not implemented")
)

// Controller represents a controller.
type Controller struct {
	Config
}

// Config represents a controller configuration.
type Config struct {
	Cloud    cloudprovider.Interface
	UserData *userdata.UserData
}

// Validate validates controller configuration.
func (c *Config) Validate() error {
	// TODO: add more validation and probably remove below IsRegistered check
	if ok := cloudprovider.IsRegistered(c.Cloud.ProviderName()); !ok {
		return fmt.Errorf("unknown cloud provider: %q", c.Cloud.ProviderName())
	}
	return nil
}

// New creates a new controller instance given a cfg config.
func New(cfg Config) *Controller {
	return &Controller{Config: cfg}
}

// CreateCluster creates a new cluster, which includes master node pool and
// other supported resources that make up a cluster.
func (c *Controller) CreateCluster(cluster model.Cluster) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}
	lb, impl := c.Cloud.LoadBalancer()
	if !impl {
		return ErrNotImplemented
	}

	if err := cl.CreateCluster(cluster); err != nil {
		return err
	}

	if err := c.CreateMasterPool(cluster.MasterPool); err != nil {
		return err
	}

	if err := lb.CreateLoadBalancer(cluster.MasterPool); err != nil {
		return err
	}

	// TODO Create DNS records
	// TODO Create compute pool
	return nil
}

// CreateMasterPool create a compute node pool.
func (c *Controller) CreateMasterPool(p model.MasterPool) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	ips, err := cl.GetMasterPersistentIPs(p.ClusterName)
	if err != nil {
		return err
	}

	// TODO(vaijab): get kube API url
	cloudConfig, err := c.UserData.RenderMasterCloudConfig(p.ClusterName, p.KubeVersion, "", ips)
	if err != nil {
		return err
	}
	p.UserData = cloudConfig

	if err := pooler.CreateMasterPool(p); err != nil {
		return err
	}
	return nil
}

// CreateComputePool create a compute node pool.
func (c *Controller) CreateComputePool(p model.ComputePool) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	err := pooler.CreateComputePool(p)
	if err != nil {
		return err
	}
	return nil
}

// ListNodePools returns a list of node pools
func (c *Controller) ListNodePools(clusterName string) ([]*model.NodePool, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return []*model.NodePool{}, ErrNotImplemented
	}

	pools, err := pooler.ListNodePools(clusterName)
	if err != nil {
		return []*model.NodePool{}, err
	}
	return pools, nil
}

// DeleteNodePool deletes a node pool
func (c *Controller) DeleteNodePool(clusterName, name string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	if err := pooler.DeleteNodePool(clusterName, name); err != nil {
		return err
	}
	return nil
}
