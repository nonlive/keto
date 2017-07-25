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

package model

// Assets is a representation of asset files as byte arrays.
type Assets struct {
	EtcdCACert []byte
	EtcdCAKey  []byte
	KubeCACert []byte
	KubeCAKey  []byte
}

// Cluster is a representation of a single cluster.
type Cluster struct {
	ResourceMeta
	MasterPool   MasterPool
	ComputePools []ComputePool
	DNSZone      string
	KubeAPIURL   string
	Status
}

// Labels a map of labels.
type Labels map[string]string

// Taints is map of kubelet taints.
type Taints map[string]string

// MasterPool is a representation of a master control plane node pool.
type MasterPool struct {
	NodePool
}

// ComputePool is a representation of a compute node pool.
type ComputePool struct {
	NodePool
}

// NodePool is a representation of a single node pool.
type NodePool struct {
	ResourceMeta
	NodePoolSpec
	Status
}

// NodePoolSpec represent a node pool.
type NodePoolSpec struct {
	KubeVersion   string   `json:"kube_version,omitempty"`
	MachineType   string   `json:"machine_type,omitempty"`
	CoreOSVersion string   `json:"coreos_version,omitempty"`
	SSHKey        string   `json:"ssh_key,omitempty"`
	DiskSize      int      `json:"disk_size,omitempty"`
	Size          int      `json:"size,omitempty"`
	Networks      []string `json:"networks,omitempty"`
	UserData      []byte   `json:"user_data,omitempty"`
	Taints        `json:"taints,omitempty"`
	KubeArgs      `json:"kube_args,omitempty"`
}

// ResourceMeta is a resource metadata.
type ResourceMeta struct {
	Name        string `json:"name,omitempty"`
	ID          string `json:"id,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Labels      `json:"labels,omitempty"`
	Internal    bool `json:"internal,omitempty"`
}

// Status is the observed status of a resource.
type Status struct {
	Created  int64  `json:"created,omitempty"`
	Upgraded int64  `json:"upgraded,omitempty"`
	State    string `json:"state,omitempty"`
}

// NodeData contains cloud Node related data.
type NodeData struct {
	KubeAPIURL  string
	ClusterName string
	KubeVersion string
	KubeArgs
	Labels
	Taints
}

// KubeArgs represents the optional extra flags for Kubernetes components.
type KubeArgs struct {
	KubeletExtraArgs           string
	APIServerExtraArgs         string
	ControllerManagerExtraArgs string
	SchedulerExtraArgs         string
}
