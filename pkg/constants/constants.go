package constants

const (
	// DefaultKubeVersion specifies a default kubernetes version.
	DefaultKubeVersion = "v1.6.1"
	// DefaultComputePoolSize specifies a default number of machines in a single compute pool.
	DefaultComputePoolSize = 1
	// DefaultDiskSizeInGigabytes specifies a default node disk size in gigabytes.
	DefaultDiskSizeInGigabytes = 10
	// DefaultCoreOSVersion specifies a default CoreOS version.
	DefaultCoreOSVersion = "TODO"
)

var (
	// ValidResourceTypes contains a list of currently supported resource types.
	ValidResourceTypes = []string{"cluster", "masterpool", "computepool"}
)
