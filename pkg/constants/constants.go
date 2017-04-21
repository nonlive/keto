package constants

const (
	// DefaultKubeVersion specifies a default kubernetes version.
	DefaultKubeVersion = "v1.6.1"
	// DefaultComputePoolSize specifies a default number of machines in a single compute pool.
	DefaultComputePoolSize = 1
	// DefaultDiskSizeInGigabytes specifies a default node disk size in gigabytes.
	DefaultDiskSizeInGigabytes = 10
	// DefaultCoreOSVersion specifies a default CoreOS version.
	// TODO only works for AWS cloud for now. Need to figure out some sort of
	// validation and CoreOS version to cloud image name mapping.
	DefaultCoreOSVersion = "CoreOS-beta-1325.2.0-hvm"
)

var (
	// ValidResourceTypes contains a list of currently supported resource types.
	ValidResourceTypes = []string{"cluster", "masterpool", "computepool"}
)
