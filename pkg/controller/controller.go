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
	"github.com/UKHomeOffice/keto/pkg/userdata"
)

var (
	// ErrNotImplemented is an error for not implemented features.
	ErrNotImplemented = errors.New("not implemented")
)

// Controller represents a controller.
type Controller struct {
	Config
	p cloudprovider.NodePooler
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

// CreateEtcdNodePool creates an etcd node pool.
// func (c *Controller) CreateEtcdNodePool() error {
// 	pooler, _ := c.Cloud.NodePooler()
// 	pooler.

// 	return nil
// }

// ListNodePools returns a list of node pools
func (c *Controller) ListNodePools(clusterName string) (cloudprovider.NodePools, error) {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return cloudprovider.NodePools{}, ErrNotImplemented
	}

	pools, err := pooler.ListNodePools(clusterName)
	if err != nil {
		return cloudprovider.NodePools{}, err
	}
	return pools, nil
}

// CreateNodePool create a node pool
func (c *Controller) CreateNodePool(clusterName, poolName string) error {
	pooler, impl := c.Cloud.NodePooler()
	if !impl {
		return ErrNotImplemented
	}

	p := cloudprovider.NodePool{
		Kind: "compute",
		ObjectMeta: cloudprovider.ObjectMeta{
			Name:        poolName,
			ClusterName: clusterName,
		},
	}

	err := pooler.CreateNodePool(p)
	if err != nil {
		return err
	}
	return nil
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
