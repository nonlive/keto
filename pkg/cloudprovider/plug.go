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
	"fmt"
	"io"
	"log"
	"sync"
)

var (
	providersMutex sync.Mutex
	providers      = make(map[string]Factory)
)

// Factory is a function that returns a cloudprovider.Interface. A config
// parameter provides an io.Reader handler to the factory to load optional
// cloud config.
type Factory func(config io.Reader) (Interface, error)

// Register registers a cloudprovider.Interface by name. This
// is expected to be called during main startup.
func Register(name string, cloud Factory) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	if cloud == nil {
		panic("cloudprovider: Register cloud is nil")
	}
	if _, dup := providers[name]; dup {
		log.Fatalf("Register was called twice for cloud provider %q", name)
	}
	providers[name] = cloud
}

// InitCloudProvider creates an instance of the named cloud provider. Optional
// cloud specific config can be passed in as io.Reader.
func InitCloudProvider(name string, config io.Reader) (Interface, error) {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	f, found := providers[name]
	if !found {
		return nil, fmt.Errorf("unknown cloud provider: %q", name)
	}
	// return a cloud-specific Factory result, that initializes the cloud config
	return f(config)
}

// IsRegistered returns a bool whether a given cloud provider is registered.
func IsRegistered(name string) bool {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	_, found := providers[name]
	return found
}

// CloudProviders returns a list of registered cloud providers.
func CloudProviders() []string {
	providersMutex.Lock()
	defer providersMutex.Unlock()
	names := []string{}
	for name := range providers {
		names = append(names, name)
	}
	return names
}
