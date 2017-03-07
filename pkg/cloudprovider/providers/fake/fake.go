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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	"github.com/UKHomeOffice/keto/pkg/cloudprovider"
)

// ProviderName is the name of this provider.
const ProviderName = "fake"

// Cloud is a test-double implementation of cloudprovider.Interface.
type Cloud struct {
	mutex sync.Mutex
	pools cloudprovider.NodePools
	// stateFile is a file name that is used for the cloud state persistence.
	// Generally, it won't be useful, except for when using the cli with fake
	// cloud for testing.
	stateFile string
}

// Compile-time check whether Cloud type value implements
// cloudprovider.Interface interface.
var _ cloudprovider.Interface = (*Cloud)(nil)

// NodePooler returns an implementation of NodePooler interface for
// Fake Cloud.
func (c *Cloud) NodePooler() (cloudprovider.NodePooler, bool) {
	return c, true
}

// ProviderName returns the cloud provider ID.
func (c *Cloud) ProviderName() string {
	return ProviderName
}

// CreateNodePool creates a node pool.
func (c *Cloud) CreateNodePool(nodePool cloudprovider.NodePool) error {
	if err := c.load(); err != nil {
		return err
	}
	c.mutex.Lock()
	c.pools = append(c.pools, &nodePool)
	c.mutex.Unlock()
	if err := c.save(); err != nil {
		return err
	}
	return nil
}

// ListNodePools lists node pools that belong to a given clusterName.
func (c *Cloud) ListNodePools(clusterName string) (cloudprovider.NodePools, error) {
	var pools cloudprovider.NodePools
	if err := c.load(); err != nil {
		return pools, err
	}
	c.mutex.Lock()
	for _, p := range c.pools {
		if clusterName != "" {
			if p.ClusterName != clusterName {
				continue
			}
		}
		pools = append(pools, p)
	}
	c.mutex.Unlock()
	return pools, nil
}

// DescribeNodePool lists nodes pools.
func (c *Cloud) DescribeNodePool() error {
	return nil
}

// UpgradeNodePool upgrades a node pool. Fake cloud implementation deletes the
// node pool first and creates a new one.
func (c *Cloud) UpgradeNodePool(nodePool cloudprovider.NodePool) error {
	if err := c.DeleteNodePool(nodePool.ClusterName, nodePool.Name); err != nil {
		return err
	}
	if err := c.CreateNodePool(nodePool); err != nil {
		return err
	}
	return nil
}

// DeleteNodePool deletes a node pool.
func (c *Cloud) DeleteNodePool(clusterName string, name string) error {
	if err := c.load(); err != nil {
		return err
	}
	for i, p := range c.pools {
		if p.ClusterName == clusterName && p.Name == name {
			// Remove items from the slice that match above condition
			c.mutex.Lock()
			c.pools = append(c.pools[:i], c.pools[i+1:]...)
			c.mutex.Unlock()
		}
	}
	if err := c.save(); err != nil {
		return err
	}
	return nil
}

// init registers Fake cloud with the cloudprovider.
func init() {
	// f knows how to initialize the cloud with given config
	f := func(config io.Reader) (cloudprovider.Interface, error) {
		return newCloud(config)
	}
	cloudprovider.Register(ProviderName, f)
}

// newCloud is used when testing Fake cloud implementation. It takes an
// optional io.Reader argument and expects it to contain state file name.
func newCloud(config io.Reader) (*Cloud, error) {
	c := &Cloud{}
	sf, stateFileEnvSet := os.LookupEnv("KETO_FAKE_STATE_FILE")
	if stateFileEnvSet {
		c.stateFile = sf
	}

	// load state from c.stateFile
	if err := c.load(); err != nil {
		return c, fmt.Errorf("invalid state file: %v", err)
	}
	return c, nil
}

func (c *Cloud) load() error {
	// in-memory state
	if c.stateFile == "" {
		return nil
	}

	// ignore non-existent or empty state files
	fileInfo, err := os.Stat(c.stateFile)
	if os.IsNotExist(err) || fileInfo.Size() == 0 {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	b, err := ioutil.ReadFile(c.stateFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &c.pools); err != nil {
		return err
	}

	return nil
}

func (c *Cloud) save() error {
	// in-memory state
	if c.stateFile == "" {
		return nil
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	b, err := json.MarshalIndent(c.pools, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(c.stateFile, b, 0666)
	if err != nil {
		return err
	}

	return nil
}
