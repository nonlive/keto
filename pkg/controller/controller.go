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
	// ErrClusterAlreadyExists is an error to report an existing cluster.
	ErrClusterAlreadyExists = errors.New("cluster already exists")
	// ErrClusterDoesNotExist is an error to report a non-existing cluster.
	ErrClusterDoesNotExist = errors.New("cluster does not exist")
	// ErrMasterPoolAlreadyExists is an error to report an existing master pool.
	ErrMasterPoolAlreadyExists = errors.New("masterpool already exists")
	// ErrComputePoolAlreadyExists is an error to report an existing compute pool.
	ErrComputePoolAlreadyExists = errors.New("computepool already exists")
)

// Controller represents a controller.
type Controller struct {
	Config
}

// Config represents a controller configuration.
type Config struct {
	Logger   logger
	Cloud    cloudprovider.Interface
	UserData userdata.UserDater
}

// logger is a generic interface that is used for passing in a logger.
type logger interface {
	Printf(string, ...interface{})
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

	c.Logger.Printf("checking whether cluster %q already exists", cluster.Name)
	exists, err := c.clusterExists(cluster.Name, cl)
	if err != nil {
		return err
	}
	if exists {
		return ErrClusterAlreadyExists
	}
	c.Logger.Printf("cluster %q does not exist", cluster.Name)

	// Both internal and external pools aren't supported at the same time.
	// See https://github.com/UKHomeOffice/keto/issues/71
	if cluster.Internal {
		c.Logger.Printf("cluster is internal, node pools will also be internal")
	}

	// Initialize Labels map in case it hasn't been.
	if cluster.Labels == nil {
		cluster.Labels = model.Labels{}
	}
	// Set default cluster labels.
	cluster.Labels[constants.ClusterNameLabelKey] = cluster.Name

	c.Logger.Printf("creating cluster %q infrastructure", cluster.Name)
	if err := cl.CreateClusterInfra(cluster); err != nil {
		return err
	}

	c.Logger.Printf("pushing cluster %q assets", cluster.Name)
	if err := cl.PushAssets(cluster.Name, assets); err != nil {
		return err
	}

	c.Logger.Printf("creating masterpool %q in cluster %q", cluster.MasterPool.Name, cluster.Name)
	if err := c.CreateMasterPool(cluster.MasterPool); err != nil {
		return err
	}

	// A user may decide not to create a compute pool during a cluster creation.
	if len(cluster.ComputePools) > 0 {
		for i := 0; i < len(cluster.ComputePools); i++ {
			c.Logger.Printf("creating computepool %q in cluster %q", cluster.ComputePools[i].Name, cluster.Name)
			if err := c.CreateComputePool(cluster.ComputePools[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateMasterPool creates a master node pool.
func (c *Controller) CreateMasterPool(p model.MasterPool) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	clusters, err := c.GetClusters(p.ClusterName)
	if err != nil {
		return err
	}
	if len(clusters) == 0 {
		return ErrClusterDoesNotExist
	}
	if len(clusters) > 1 {
		return fmt.Errorf("more than one cluster found matching %q name", p.ClusterName)
	}
	p.Internal = clusters[0].Internal

	c.Logger.Printf("checking whether masterpool %q already exists in cluster %q", p.Name, p.ClusterName)
	m, err := c.GetMasterPools(p.ClusterName, "")
	if err != nil {
		return err
	}
	if len(m) != 0 {
		return ErrMasterPoolAlreadyExists
	}
	c.Logger.Printf("masterpool %q does not exist in cluster %q", p.Name, p.ClusterName)

	// Use defaults if values aren't specified.
	if p.DiskSize == 0 {
		p.DiskSize = constants.DefaultDiskSizeInGigabytes
		c.Logger.Printf("disk size is not specified, using default %d", p.DiskSize)
	}
	if p.KubeVersion == "" {
		p.KubeVersion = constants.DefaultKubeVersion
		c.Logger.Printf("kube version is not specified, using default %q", p.KubeVersion)
	}
	if p.CoreOSVersion == "" {
		p.CoreOSVersion = constants.DefaultCoreOSVersion
		c.Logger.Printf("coreos version is not specified, using default %q", p.CoreOSVersion)
	}

	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	c.Logger.Printf("getting master persistent IP addresses and their IDs for cluster %q", p.ClusterName)
	ips, err := cl.GetMasterPersistentIPs(p.ClusterName)
	if err != nil {
		return err
	}
	c.Logger.Printf("got IPs and IDs: %#v", ips)

	cloudConfig, err := c.UserData.RenderMasterCloudConfig(c.Cloud.ProviderName(), p.ClusterName, p.KubeVersion, ips)
	if err != nil {
		return err
	}
	p.UserData = cloudConfig

	// Cluster scope labels get applied to node pools by default.
	if p.Labels == nil {
		p.Labels = model.Labels{}
	}
	for k, v := range clusters[0].Labels {
		p.Labels[k] = v
	}
	p.Labels[constants.PoolNameLabelKey] = p.Name

	return pooler.CreateMasterPool(p)
}

func (c *Controller) clusterExists(name string, cl cloudprovider.Clusters) (bool, error) {
	clusters, err := cl.GetClusters(name)
	if err != nil || len(clusters) != 1 {
		return false, nil
	}
	return true, nil
}

// CreateComputePool create a compute node pool.
func (c *Controller) CreateComputePool(p model.ComputePool) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	clusters, err := c.GetClusters(p.ClusterName)
	if err != nil {
		return err
	}
	if len(clusters) != 1 {
		return ErrClusterDoesNotExist
	}
	if len(clusters) > 1 {
		return fmt.Errorf("more than one cluster found matching %q name", p.ClusterName)
	}
	p.Internal = clusters[0].Internal

	// Check if a compute pool with the same name exists already.
	c.Logger.Printf("checking whether computepool %q already exists in cluster %q", p.Name, p.ClusterName)
	computeExists, err := c.computePoolExists(p.ClusterName, p.Name, pooler)
	if err != nil {
		return err
	}
	if computeExists {
		return ErrComputePoolAlreadyExists
	}
	c.Logger.Printf("computepool %q does not exist in cluster %q", p.Name, p.ClusterName)

	// Use defaults if values aren't specified.
	if p.DiskSize == 0 {
		p.DiskSize = constants.DefaultDiskSizeInGigabytes
		c.Logger.Printf("disk size is not specified, using default %d", p.DiskSize)
	}
	if p.Size == 0 {
		p.Size = constants.DefaultComputePoolSize
		c.Logger.Printf("compute pool size is not specified, using default %d", p.Size)
	}

	// TODO get the missing properties from the masterpool. If not specified,
	// use versions that the masterpool is using? On the other hand, how can we
	// ensure that those versions will work with keto cli version? Maybe use
	// keto defaults instead?
	if p.KubeVersion == "" {
		p.KubeVersion = constants.DefaultKubeVersion
		c.Logger.Printf("kube version is not specified, using default %q", p.KubeVersion)
	}
	if p.CoreOSVersion == "" {
		p.CoreOSVersion = constants.DefaultCoreOSVersion
		c.Logger.Printf("coreos version is not specified, using default %q", p.CoreOSVersion)
	}

	cloudConfig, err := c.UserData.RenderComputeCloudConfig(c.Cloud.ProviderName(), p.ClusterName, p.KubeVersion)
	if err != nil {
		return err
	}
	p.UserData = cloudConfig

	// Cluster scope labels get applied to node pools by default.
	if p.Labels == nil {
		p.Labels = model.Labels{}
	}
	for k, v := range clusters[0].Labels {
		p.Labels[k] = v
	}
	p.Labels[constants.PoolNameLabelKey] = p.Name

	return pooler.CreateComputePool(p)
}

func (c *Controller) computePoolExists(clusterName, name string, pooler cloudprovider.NodePooler) (bool, error) {
	p, err := pooler.GetComputePools(clusterName, name)
	if err != nil || len(p) == 0 {
		return false, err
	}
	return true, nil
}

// GetMasterPools returns a list of master pools
func (c *Controller) GetMasterPools(clusterName string, names ...string) ([]*model.MasterPool, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return []*model.MasterPool{}, ErrNotImplemented
	}

	c.Logger.Printf("getting masterpool in cluster %q", clusterName)

	p, err := pooler.GetMasterPools(clusterName, "")
	if err != nil {
		return []*model.MasterPool{}, err
	}

	return filterMasterPools(p, names), nil
}

// GetComputePools returns a list of compute node pools.
func (c *Controller) GetComputePools(clusterName string, names ...string) ([]*model.ComputePool, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return []*model.ComputePool{}, ErrNotImplemented
	}

	c.Logger.Printf("getting computepool in cluster %q", clusterName)

	p, err := pooler.GetComputePools(clusterName, "")
	if err != nil {
		return []*model.ComputePool{}, err
	}

	return filterComputePools(p, names), nil
}

// GetClusters gets a list of clusters.
func (c *Controller) GetClusters(names ...string) ([]*model.Cluster, error) {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return []*model.Cluster{}, ErrNotImplemented
	}
	c.Logger.Printf("getting clusters")

	clusters, err := cl.GetClusters("")
	if err != nil {
		return []*model.Cluster{}, err
	}

	return filterClusters(clusters, names), nil

}

// DeleteCluster deletes a cluster.
func (c *Controller) DeleteCluster(names ...string) error {
	cl, impl := c.Cloud.Clusters()
	if !impl {
		return ErrNotImplemented
	}

	for _, n := range names {
		c.Logger.Printf("deleting cluster %q", n)
		err := cl.DeleteCluster(n)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteMasterPool deletes a master node pool.
func (c *Controller) DeleteMasterPool(clusterName string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	c.Logger.Printf("deleting masterpool of cluster %q", clusterName)
	return pooler.DeleteMasterPool(clusterName)
}

// DeleteComputePool deletes a compute node pool.
func (c *Controller) DeleteComputePool(clusterName string, names ...string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	for _, name := range names {
		c.Logger.Printf("deleting computepool %q of cluster %q", name, clusterName)
		err := pooler.DeleteComputePool(clusterName, name)
		if err != nil {
			return err
		}
	}
	return nil
}

func filterMasterPools(pools []*model.MasterPool, names []string) []*model.MasterPool {
	filteredPools := []*model.MasterPool{}

	for _, p := range pools {
		if stringInSlice(p.Name, names) {
			filteredPools = append(filteredPools, p)
		}
	}

	if len(filteredPools) != 0 {
		return filteredPools
	}
	return pools

}

func filterComputePools(pools []*model.ComputePool, names []string) []*model.ComputePool {

	filteredPools := []*model.ComputePool{}

	for _, p := range pools {
		if stringInSlice(p.Name, names) {
			filteredPools = append(filteredPools, p)
		}
	}

	if len(filteredPools) != 0 {
		return filteredPools
	}

	return pools
}

func filterClusters(clusters []*model.Cluster, names []string) []*model.Cluster {
	filteredClusters := []*model.Cluster{}

	for _, c := range clusters {
		if stringInSlice(c.Name, names) {
			filteredClusters = append(filteredClusters, c)
		}
	}

	if len(filteredClusters) != 0 {
		return filteredClusters
	}

	return clusters
}

func stringInSlice(name string, names []string) bool {
	for _, n := range names {
		if name == n {
			return true
		}
	}
	return false
}
