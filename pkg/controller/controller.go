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
	"github.com/UKHomeOffice/keto/pkg/constants"
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

	if err := cl.CreateClusterInfra(cluster); err != nil {
		return err
	}

	if err := cl.PushAssets(cluster.Name, assets); err != nil {
		return err
	}

	if err := c.CreateMasterPool(cluster.MasterPool); err != nil {
		return err
	}

	// A user may decide not to create compute pools as part of cluster creation.
	if len(cluster.ComputePools) == 1 {
		if err := c.CreateComputePool(cluster.ComputePools[0]); err != nil {
			return err
		}
	}

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
	m, err := c.GetMasterPools(p.ClusterName, "")
	if err != nil {
		return err
	}
	if len(m) != 0 {
		return fmt.Errorf("masterpool already exists in cluster %q", p.ClusterName)
	}

	// Use defaults if values aren't specified.
	if p.DiskSize == 0 {
		p.DiskSize = constants.DefaultDiskSizeInGigabytes
	}
	if p.KubeVersion == "" {
		p.KubeVersion = constants.DefaultKubeVersion
	}
	if p.CoreOSVersion == "" {
		p.CoreOSVersion = constants.DefaultCoreOSVersion
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
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	// TODO Check if a masterpool for given cluster exists first? Maybe we
	// don't care if masterpool does not exist yet, as long as the cluster infra exists?
	if exists, err := c.clusterExists(p.ClusterName, cl); !exists {
		return err
	}

	// Check if a compute pool with the same name exists already.
	computeExists, err := c.computePoolExists(p.ClusterName, p.Name, pooler)
	if err != nil {
		return err
	}
	if computeExists {
		return fmt.Errorf("compute pool %q already exists in cluster %q", p.Name, p.ClusterName)
	}

	// Use defaults if values aren't specified.
	if p.DiskSize == 0 {
		p.DiskSize = constants.DefaultDiskSizeInGigabytes
	}
	if p.Size == 0 {
		p.Size = constants.DefaultComputePoolSize
	}

	// TODO get the missing properties from the masterpool. If not specified,
	// use versions that the masterpool is using? On the other hand, how can we
	// ensure that those versions will work with keto cli version? Maybe use
	// keto defaults instead?
	if p.KubeVersion == "" {
		p.KubeVersion = constants.DefaultKubeVersion
	}
	if p.CoreOSVersion == "" {
		p.CoreOSVersion = constants.DefaultCoreOSVersion
	}

	cloudConfig, err := c.UserData.RenderComputeCloudConfig(c.Cloud.ProviderName(), p.ClusterName, p.KubeVersion)
	if err != nil {
		return err
	}
	p.UserData = cloudConfig

	return pooler.CreateComputePool(p)
}

func (c *Controller) computePoolExists(clusterName, name string, pooler cloudprovider.NodePooler) (bool, error) {
	p, err := pooler.GetComputePools(clusterName, name)
	if len(p) != 0 && err == nil {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("unable to get compute pools: %v", err)
	}
	return false, nil
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

// GetComputePools returns a list of compute node pools.
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

// DeleteCluster deletes a node pool.
func (c *Controller) DeleteCluster(name string) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	return cl.DeleteCluster(name)
}

// DeleteMasterPool deletes a node pool.
func (c *Controller) DeleteMasterPool(clusterName string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	return pooler.DeleteMasterPool(clusterName)
}

// DeleteComputePool deletes a node pool.
func (c *Controller) DeleteComputePool(clusterName, name string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	return pooler.DeleteComputePool(clusterName, name)
}
