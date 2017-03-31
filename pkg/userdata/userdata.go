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

package userdata

import (
	"bytes"
	"text/template"
)

// UserData defines a user data struct
type UserData struct {
}

// New returns a new UserData struct
func New() *UserData {
	return &UserData{}
}

// RenderMasterCloudConfig renders a master cloud-config.
func (u UserData) RenderMasterCloudConfig(
	clusterName string,
	kubeVersion string,
	kubeAPIURL string,
	masterPersistentNodeIDIP map[string]string,
) ([]byte, error) {

	const masterTemplate = `#cloud-config
coreos:
  update:
    reboot-strategy: "off"
# TODO {{ .MasterPersistentNodeIDIP }}
`

	data := struct {
		ClusterName              string
		KubeVersion              string
		KubeAPIURL               string
		MasterPersistentNodeIDIP map[string]string
	}{
		ClusterName:              clusterName,
		KubeVersion:              kubeVersion,
		KubeAPIURL:               kubeAPIURL,
		MasterPersistentNodeIDIP: masterPersistentNodeIDIP,
	}

	t := template.Must(template.New("master-cloud-config").Parse(masterTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return b.Bytes(), err
	}

	return b.Bytes(), nil
}

// RenderComputeCloudConfig renders a compute cloud-config.
func (u UserData) RenderComputeCloudConfig(kubeVersion, kubeAPIURL string) ([]byte, error) {
	const computeTemplate = `#cloud-config
coreos:
  update:
    reboot-strategy: "off"
# TODO
`

	data := struct {
		KubeVersion string
		KubeAPIURL  string
	}{
		KubeVersion: kubeVersion,
		KubeAPIURL:  kubeAPIURL,
	}

	t := template.Must(template.New("compute-cloud-config").Parse(computeTemplate))
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return b.Bytes(), err
	}

	return b.Bytes(), nil
}
