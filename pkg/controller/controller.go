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
func (c *Controller) CreateCluster(cluster model.Cluster, assets model.Assets) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	if err := cl.CreateCluster(cluster); err != nil {
		return err
	}

	if err := cl.PushAssets(assets); err != nil {
		return err
	}

	if err := c.CreateMasterPool(cluster.MasterPool); err != nil {
		return err
	}

	// TODO Create compute pool
	return nil
}

// CreateMasterPool create a master node pool.
func (c *Controller) CreateMasterPool(p model.MasterPool) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	// Check if cluster exists first.
	if exists, err := c.clusterExists(p.ClusterName, cl); !exists {
		return err
	}

	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}
	ips, err := cl.GetMasterPersistentIPs(p.ClusterName)
	if err != nil {
		return err
	}

	cloudConfig, err := c.UserData.RenderMasterCloudConfig(c.Cloud.ProviderName(), p.ClusterName, p.KubeVersion, ips)
	if err != nil {
		return err
	}
	p.UserData = cloudConfig

	return pooler.CreateMasterPool(p)
}

func (c *Controller) clusterExists(name string, cl cloudprovider.Clusters) (bool, error) {
	if c, err := cl.GetClusters(name); len(c) == 0 {
		errMsg := "cluster not found"
		if err != nil {
			errMsg = fmt.Sprintf("%s: %v", errMsg, err)
		}
		return false, errors.New(errMsg)
	}
	return true, nil
}

// CreateComputePool create a compute node pool.
func (c *Controller) CreateComputePool(p model.ComputePool) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}
	return pooler.CreateComputePool(p)
}

// GetMasterPools returns a list of master pools
func (c *Controller) GetMasterPools(clusterName, name string) ([]*model.MasterPool, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return []*model.MasterPool{}, ErrNotImplemented
	}

	pools, err := pooler.GetMasterPools(clusterName, name)
	if err != nil {
		return []*model.MasterPool{}, err
	}
	return pools, nil
}

// GetComputePools returns a list of compute node pools
func (c *Controller) GetComputePools(clusterName, name string) ([]*model.ComputePool, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return []*model.ComputePool{}, ErrNotImplemented
	}

	pools, err := pooler.GetComputePools(clusterName, name)
	if err != nil {
		return []*model.ComputePool{}, err
	}
	return pools, nil
}

// GetClusters gets a list of clusters.
func (c *Controller) GetClusters(name string) ([]*model.Cluster, error) {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return []*model.Cluster{}, ErrNotImplemented
	}
	return cl.GetClusters(name)
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
