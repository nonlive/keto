package util

import (
	"github.com/spf13/cobra"
)

// StringInSlice returns true if given slice contains s string.
func StringInSlice(s string, slice []string) bool {
	for _, i := range slice {
		if s == i {
			return true
		}
	}
	return false
}

// GetStringFlagValueIfSet returns value of the string flag if the flag has
// been actually set.
func GetStringFlagValueIfSet(c *cobra.Command, name string) string {
	var v string
	if c.Flags().Changed(name) {
		v, _ = c.Flags().GetString(name)
	}
	return v
}

// GetIntFlagValueIfSet returns value of the int flag if the flag has
// been actually set.
func GetIntFlagValueIfSet(c *cobra.Command, name string) int {
	var v int
	if c.Flags().Changed(name) {
		v, _ = c.Flags().GetInt(name)
	}
	return v
}
