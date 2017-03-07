package version

var (
	gitVersion   string
	gitCommit    string
	gitTreeState = "not a git tree" // either "clean" or "dirty"
)

// Version represents version data.
type Version struct {
	Version      string
	Commit       string
	GitTreeState string
}

// Get returns the overall codebase version.
func Get() Version {
	return Version{
		Version:      gitVersion,
		Commit:       gitCommit,
		GitTreeState: gitTreeState,
	}
}
