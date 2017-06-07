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

//go:generate mockery -dir $GOPATH/src/github.com/UKHomeOffice/keto/pkg/userdata -name=UserDater

package userdata

import (
	"testing"

	"github.com/UKHomeOffice/keto/testutil"
)

const clusterName = "foo"

func TestRenderMasterCloudConfig(t *testing.T) {
	u := New()
	s, err := u.RenderMasterCloudConfig("aws", clusterName, "v1.7.0", map[string]string{"0": "10.0.0.1"})
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, string(s), clusterName)
}

func TestRenderComputeCloudConfig(t *testing.T) {
	u := New()
	s, err := u.RenderComputeCloudConfig("aws", clusterName, "v1.7.0")
	if err != nil {
		t.Error(err)
	}
	testutil.CheckTemplate(t, string(s), clusterName)
}
