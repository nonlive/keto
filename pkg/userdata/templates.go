package userdata

var (
	// EtcdTemplate is userdata template for etcd node pool
	EtcdTemplate = []byte(`#cloud-config
coreos:
  # TODO`)

	// MasterTemplate is userdata template for master node pool
	MasterTemplate = []byte(`#cloud-config
coreos:
  # TODO`)

	// ComputeTemplate userdata template for compute node pool
	ComputeTemplate = []byte(`#cloud-config
coreos:
  # TODO`)
)
