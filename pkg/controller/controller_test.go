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
	"log"
	"os"
	"testing"

	cloudProviderMocks "github.com/UKHomeOffice/keto/pkg/cloudprovider/mocks"
	userdataMocks "github.com/UKHomeOffice/keto/pkg/userdata/mocks"

	"github.com/UKHomeOffice/keto/pkg/model"
	"github.com/UKHomeOffice/keto/testutil"
)

const cloudProviderName = "mock"

type testMock struct {
	Provider   *cloudProviderMocks.Interface
	Clusters   *cloudProviderMocks.Clusters
	NodePooler *cloudProviderMocks.NodePooler
	Node       *cloudProviderMocks.Node
	UserData   *userdataMocks.UserDater
}

func TestCreateCluster(t *testing.T) {
	m, ctrl := makeTestMock()

	persistentIPs := map[string]string{"node0": "1.1.1.1"}
	cluster := model.Cluster{
		Name:       "foo",
		MasterPool: model.MasterPool{NodePool: testutil.MakeNodePool("foo", "master")},
	}

	m.Clusters.On("GetClusters", cluster.Name).Return([]*model.Cluster{}, nil).Once()
	m.Clusters.On("CreateClusterInfra", cluster).Return(nil)
	m.Clusters.On("PushAssets", cluster.Name, model.Assets{}).Return(nil)

	// At this point the cluster infra already exists.
	m.Clusters.On("GetClusters", cluster.Name).Return([]*model.Cluster{&cluster}, nil).Once()
	m.NodePooler.On("GetMasterPools", cluster.Name, "").Return([]*model.MasterPool{}, nil)
	m.Clusters.On("GetMasterPersistentIPs", cluster.Name).Return(persistentIPs, nil)
	m.Provider.On("ProviderName").Return(cloudProviderName)

	m.UserData.On("RenderMasterCloudConfig",
		cloudProviderName,
		cluster.Name,
		cluster.MasterPool.KubeVersion,
		persistentIPs).Return(cluster.MasterPool.UserData,
		nil)

	m.NodePooler.On("CreateMasterPool", cluster.MasterPool).Return(nil)

	if err := ctrl.CreateCluster(cluster, model.Assets{}); err != nil {
		t.Error(err)
	}

	m.Clusters.AssertExpectations(t)
	m.UserData.AssertExpectations(t)
	m.NodePooler.AssertExpectations(t)
	m.Provider.AssertExpectations(t)
}

func TestCreateClusterAlreadyExists(t *testing.T) {
	m, ctrl := makeTestMock()

	cluster := model.Cluster{
		Name:       "foo",
		MasterPool: model.MasterPool{NodePool: testutil.MakeNodePool("foo", "master")},
	}

	m.Clusters.On("GetClusters", cluster.Name).Return([]*model.Cluster{&cluster}, nil).Once()

	if err := ctrl.CreateCluster(cluster, model.Assets{}); err != ErrClusterAlreadyExists {
		t.Errorf("wrong error; got %q; want %q", err, ErrClusterAlreadyExists)
	}

	m.Clusters.AssertExpectations(t)
}

func TestCreateMasterPoolAlreadyExists(t *testing.T) {
	m, ctrl := makeTestMock()

	clusterName := "foo"
	p := model.MasterPool{
		NodePool: testutil.MakeNodePool(clusterName, "master"),
	}

	m.Clusters.On("GetClusters", clusterName).Return([]*model.Cluster{&model.Cluster{Name: clusterName}}, nil).Once()
	m.NodePooler.On("GetMasterPools", clusterName, "").Return([]*model.MasterPool{&p}, nil)

	if err := ctrl.CreateMasterPool(p); err != ErrMasterPoolAlreadyExists {
		t.Errorf("wrong error; got %q; want %q", err, ErrMasterPoolAlreadyExists)
	}

	m.Clusters.AssertExpectations(t)
	m.NodePooler.AssertExpectations(t)
}

func TestDeleteCluster(t *testing.T) {
	m, ctrl := makeTestMock()
	m.Clusters.On("DeleteCluster", "foo").Return(nil)

	if err := ctrl.DeleteCluster("foo"); err != nil {
		t.Error(err)
	}

	m.Clusters.AssertExpectations(t)
}

func makeTestMock() (*testMock, *Controller) {
	m := &testMock{
		Provider:   &cloudProviderMocks.Interface{},
		Clusters:   &cloudProviderMocks.Clusters{},
		NodePooler: &cloudProviderMocks.NodePooler{},
		Node:       &cloudProviderMocks.Node{},
		UserData:   &userdataMocks.UserDater{},
	}

	m.Provider.On("Clusters").Return(m.Clusters, true)
	m.Provider.On("NodePooler").Return(m.NodePooler, true)

	ctrl := New(Config{
		Logger:   log.New(os.Stderr, "", log.LstdFlags),
		Cloud:    m.Provider,
		UserData: m.UserData,
	})
	return m, ctrl
}
