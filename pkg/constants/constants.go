package constants

const (
	// DefaultKubeVersion specifies a default kubernetes version.
	DefaultKubeVersion = "v1.7.0"
	// DefaultNetworkProvider specifies what CNI provider to install
	DefaultNetworkProvider = "canal"
	// DefaultKetoK8Image specifies the image to use for keto-k8 container
	DefaultKetoK8Image = "quay.io/ukhomeofficedigital/keto-k8:v0.1.1-b1"
	// DefaultComputePoolSize specifies a default number of machines in a single compute pool.
	DefaultComputePoolSize = 1
	// DefaultDiskSizeInGigabytes specifies a default node disk size in gigabytes.
	DefaultDiskSizeInGigabytes = 10
	// DefaultCoreOSVersion specifies a default CoreOS version.
	// TODO only works for AWS cloud for now. Need to figure out some sort of
	// validation and CoreOS version to cloud image name mapping.
	DefaultCoreOSVersion = "CoreOS-stable-1353.8.0-hvm"

	// ClusterNameLabelKey label key name for cluster name label.
	ClusterNameLabelKey = "cluster-name"
	// PoolNameLabelKey label key name for pool name label.
	PoolNameLabelKey = "pool-name"
)
