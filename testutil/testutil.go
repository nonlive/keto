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

// Package testutil provides test utility functions.
package testutil

import (
	"strings"
	"testing"

	"github.com/UKHomeOffice/keto/pkg/model"
)

// CheckTemplate is a testing helper function that attempts to find match
// string in s string.
func CheckTemplate(t *testing.T, s, match string) {
	if !strings.Contains(s, match) {
		t.Errorf("failed to render the template; %q not found", match)
	}
}

// MakeNodePool is a helper function that makes a new model.NodePool for testing.
func MakeNodePool(clusterName, name string) model.NodePool {
	meta := model.ResourceMeta{
		Name:        name,
		ClusterName: clusterName,
	}

	spec := model.NodePoolSpec{
		KubeVersion:   "v1.7.0",
		MachineType:   "tiny",
		CoreOSVersion: "CoreOS-beta-1409.1.0-hvm",
		SSHKey:        "s3cr3tkey",
		DiskSize:      10,
		Size:          1,
		Networks:      []string{"network0", "network1"},
		UserData:      []byte("mocked userdata"),
	}

	return model.NodePool{
		ResourceMeta: meta,
		NodePoolSpec: spec,
	}
}
