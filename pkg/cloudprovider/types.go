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

// NodePools is a NodePool type slice.
type NodePools []*NodePool

// NodePool is a representation of a single node pool.
type NodePool struct {
	Kind string `json:"kind,omitempty"`
	ObjectMeta
	NodePoolSpec
	ObjectStatus
}

// ObjectMeta is a resource metadata.
type ObjectMeta struct {
	Name        string            `json:"name,omitempty"`
	ClusterName string            `json:"cluster_name,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ObjectStatus is the observed status of a cloud resource.
type ObjectStatus struct {
	Created  int64  `json:"created,omitempty"`
	Upgraded int64  `json:"upgraded,omitempty"`
	State    string `json:"state,omitempty"`
}

// NodePoolSpec is a node pool spec that is used when creating/upgrading a node
// pool.
type NodePoolSpec struct {
	InstanceType string   `json:"instance_type,omitempty"`
	OSVersion    string   `json:"os_version,omitempty"`
	MinSize      int      `json:"min_size,omitempty"`
	Networks     []string `json:"networks,omitempty"`
	UserData     []byte   `json:"user_data,omitempty"`
}

// type Instances []*Instance

// type Instances struct {
// }

// Clusters is a Cluster type slice.
// type Clusters []*Cluster

// Cluster is a reprenstation of a single cluster.
// type Cluster struct {
// 	Name string
// 	NodePools
// }
