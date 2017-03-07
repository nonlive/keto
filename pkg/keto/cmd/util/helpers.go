package util

// StringInSlice returns true if given slice contains s string.
func StringInSlice(s string, slice []string) bool {
	for _, i := range slice {
		if s == i {
			return true
		}
	}
	return false
}
