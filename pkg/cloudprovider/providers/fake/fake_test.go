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

package fake

import (
	"testing"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
)

func TestCreateNodePool(t *testing.T) {
	fc, err := newCloud(nil)
	if err != nil {
		t.Fatalf("failed to initialize fake cloud: %v", err)
	}

	pools := []cloudprovider.NodePool{
		{
			Kind: "etcd",
			ObjectMeta: cloudprovider.ObjectMeta{
				Name:        "etcd",
				ClusterName: "test",
			},
			NodePoolSpec: cloudprovider.NodePoolSpec{
				InstanceType: "medium",
				OSVersion:    "1234.0.0",
			},
		},
		{
			Kind: "master",
			ObjectMeta: cloudprovider.ObjectMeta{
				Name:        "master",
				ClusterName: "test",
			},
			NodePoolSpec: cloudprovider.NodePoolSpec{
				InstanceType: "medium",
				OSVersion:    "1234.0.0",
			},
		},
		{
			Kind: "compute",
			ObjectMeta: cloudprovider.ObjectMeta{
				Name:        "compute",
				ClusterName: "test",
			},
			NodePoolSpec: cloudprovider.NodePoolSpec{
				InstanceType: "medium",
				OSVersion:    "1234.0.0",
			},
		},
	}

	for _, p := range pools {
		if err := fc.CreateNodePool(p); err != nil {
			t.Fatalf("Should be able to create %q node pool: %v", p.Name, err)
		}
	}
}

var testPool = cloudprovider.NodePool{
	Kind: "compute",
	ObjectMeta: cloudprovider.ObjectMeta{
		Name:        "mypool",
		ClusterName: "mycluster",
	},
	NodePoolSpec: cloudprovider.NodePoolSpec{
		InstanceType: "medium",
		OSVersion:    "1234.0.0",
	},
}

func TestListNodePools(t *testing.T) {
	fc, err := newCloud(nil)
	if err != nil {
		t.Fatalf("failed to initialize fake cloud: %v", err)
	}

	if err := fc.CreateNodePool(testPool); err != nil {
		t.Fatalf("Should be able to create a node pool: %v", err)
	}
	for _, clusterName := range []string{"", testPool.ClusterName} {
		pools, err := fc.ListNodePools(clusterName)
		if err != nil {
			t.Errorf("Should be able to list node pools: %v", err)
		}
		if len(pools) == 1 {
			return
		}
		t.Errorf("Expected one node pool from %q cluster to be returned. Instead got: %q", clusterName, pools)
	}
}

func TestDeleteNodePool(t *testing.T) {
	fc, err := newCloud(nil)
	if err != nil {
		t.Fatalf("failed to initialize fake cloud: %v", err)
	}

	if err := fc.CreateNodePool(testPool); err != nil {
		t.Fatalf("Should be able to create a node pool: %v", err)
	}
	err = fc.DeleteNodePool(testPool.ClusterName, testPool.Name)
	if err != nil {
		t.Errorf("Should be able to delete a node pool: %v", err)
	}
	pools, err := fc.ListNodePools(testPool.ClusterName)
	if err != nil {
		t.Errorf("Should be able to list node pools: %v", err)
	}
	if len(pools) == 1 {
		t.Errorf("Expected no node pools from %q cluster to be returned. Instead got: %v", testPool.ClusterName, pools)
	}
}

func TestUpgradeNodePool(t *testing.T) {
	fc, err := newCloud(nil)
	if err != nil {
		t.Fatalf("failed to initialize fake cloud: %v", err)
	}

	want := "1234.1.0"
	if err := fc.CreateNodePool(testPool); err != nil {
		t.Fatalf("Should be able to create a node pool: %v", err)
	}

	upgradeTo := testPool
	upgradeTo.OSVersion = want
	if err := fc.UpgradeNodePool(upgradeTo); err != nil {
		t.Errorf("error upgrading a node pool: %v", err)
	}

	// TODO should really use DescribeNodePool()
	pools, err := fc.ListNodePools(upgradeTo.ClusterName)
	if err != nil {
		t.Fatalf("error listing node pools: %v", err)
	}
	if pools[0].OSVersion != want {
		t.Errorf("Got %q, want %q", pools[0].OSVersion, want)
	}
}
