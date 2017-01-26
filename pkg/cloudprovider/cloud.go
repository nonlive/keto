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

// Interface is an abstract interface for cloud providers.
type Interface interface {
	// Clusterer returns a clusters interface. Also returns true if the
	// interface is supported, false otherwise.
	// Clusterer() (Clusterer, bool)
	// NodePoolManager returns a node pools interface. Also returns true if the
	// interface is supported, false otherwise.
	NodePooler() (NodePooler, bool)
	// ProviderName returns the cloud provider name.
	ProviderName() string
}

// Clusterer is an abstract interface for clusters of node pools.
// type Clusterer interface {
// 	// CreateCluster creates a new cluster.
// 	CreateCluster() error
// 	// ListCluster returns a list of clusters.
// 	ListClusters(clusterName string) (Clusters, error)
// 	// DescribeCluster describes a given cluster.
// 	DescribeCluster() error
// 	// UpgradeCluster upgrades nodepools in the cluster.
// 	UpgradeCluster() error
// 	// DeleteCluster deletes entire cluster.
// 	DeleteCluster() error
// }

// NodePooler is an abstract interface for node pools.
type NodePooler interface {
	// CreatePool creates a new node pool.
	CreateNodePool(nodePool NodePool) error
	// ListNodePools returns a list of node pools that are part of a given clusterName.
	ListNodePools(clusterName string) (NodePools, error)
	// DescribeNodePool describes a given node pool.
	DescribeNodePool() error
	// UpgradePool upgrades a node pool.
	UpgradeNodePool(nodePool NodePool) error
	// DeleteNodePool deletes a node pool.
	DeleteNodePool(clusterName string, name string) error
}

// BuildNodePoolName returns a node pool name.
func BuildNodePoolName(clusterName, poolName, poolKind string) string {
	var name string
	switch poolKind {
	case "etcd":
		name = clusterName + "-etcd"
	case "master":
		name = clusterName + "-master"
	default:
		name = clusterName + "-" + poolName
	}
	return name
}
