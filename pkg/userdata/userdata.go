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
	masterPersistentIPs []string,
) ([]byte, error) {

	const masterTemplate = `#cloud-config
coreos:
  update:
    reboot-strategy: "off"
# TODO {{ .MasterPersistentIPs }}
`

	data := struct {
		ClusterName         string
		KubeVersion         string
		KubeAPIURL          string
		MasterPersistentIPs []string
	}{
		ClusterName:         clusterName,
		KubeVersion:         kubeVersion,
		KubeAPIURL:          kubeAPIURL,
		MasterPersistentIPs: masterPersistentIPs,
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
