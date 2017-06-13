package constants

const (
	// DefaultKubeVersion specifies a default kubernetes version.
	DefaultKubeVersion = "v1.6.4"
	// DefaultNetworkProvider specifies what CNI provider to install
	DefaultNetworkProvider = "canal"
	// DefaultKetoK8Image specifies the image to use for keto-k8 container
	DefaultKetoK8Image = "quay.io/ukhomeofficedigital/keto-k8:v0.0.3"
	// DefaultComputePoolSize specifies a default number of machines in a single compute pool.
	DefaultComputePoolSize = 1
	// DefaultDiskSizeInGigabytes specifies a default node disk size in gigabytes.
	DefaultDiskSizeInGigabytes = 10
	// DefaultCoreOSVersion specifies a default CoreOS version.
	// TODO only works for AWS cloud for now. Need to figure out some sort of
	// validation and CoreOS version to cloud image name mapping.
	DefaultCoreOSVersion = "CoreOS-stable-1353.6.0-hvm"
	// DefaultClusterName is the cluster name
	DefaultClusterName = "cluster"
	// DefaultMasterName is the name of the masterpool prefix
	DefaultMasterName = "masterpool"
	// DefaultComputeName is the name of the masterpool prefix
	DefaultComputeName = "computepool"
)

var (
	// ValidResourceTypes contains a list of currently supported resource types.
	ValidResourceTypes = []string{DefaultClusterName, DefaultMasterName, DefaultComputeName}
)
