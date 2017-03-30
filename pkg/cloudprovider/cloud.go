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

package cloudprovider

import (
	"github.com/UKHomeOffice/keto/pkg/model"
)

// Interface is an abstract interface for cloud providers.
type Interface interface {
	// ProviderName returns the cloud provider name.
	ProviderName() string
	// Clusters returns a clusters interface. Also returns true if the
	// interface is supported, false otherwise.
	Clusters() (Clusters, bool)
	// NodePooler returns a node pools interface. Also returns true if the
	// interface is supported, false otherwise.
	NodePooler() (NodePooler, bool)
	// LoadBalancer returns a load balancer interface. Also returns true if the
	// interface is supported, false otherwise.
	LoadBalancer() (LoadBalancer, bool)
}

// Clusters is an abstract interface for clusters.
type Clusters interface {
	// CreateCluster creates a new cluster.
	CreateCluster(model.Cluster) error
	// ListClusters returns a list of clusters in the cloud account.
	ListClusters(string) ([]*model.Cluster, error)
	// DescribeCluster describes a given cluster.
	// TODO
	DescribeCluster(string) error
	// DeleteCluster deletes a cluster.
	// TODO
	DeleteCluster(string) error
	// GetMasterPersistentIPs returns a list of master persistent IPs for a given clusterName.
	GetMasterPersistentIPs(clusterName string) ([]string, error)
	// GetKubeAPIURL returns a full URL to Kubernetes API.
	GetKubeAPIURL(clusterName string) (string, error)
}

// NodePooler is an abstract interface for node pools.
type NodePooler interface {
	// CreateMasterPool creates a new master node pool.
	CreateMasterPool(pool model.MasterPool) error
	// CreateComputePool creates a new compute node pool.
	CreateComputePool(pool model.ComputePool) error
	// ListNodePools returns a list of node pools in the cloud account.
	ListNodePools(clusterName string) ([]*model.NodePool, error)
	// DescribeNodePool describes a given node pool.
	// TODO
	DescribeNodePool() error
	// UpgradePool upgrades a node pool.
	// TODO
	UpgradeNodePool() error
	// DeleteNodePool deletes a node pool.
	// TODO
	DeleteNodePool(clusterName, name string) error
}

// LoadBalancer is an abstract interface for managing load balancers.
type LoadBalancer interface {
	CreateLoadBalancer(p model.MasterPool) error
	GetLoadBalancer(clusterName string) error
	UpdateLoadBalancer() error
	DeleteLoadBalancer(clusterName string) error
}
