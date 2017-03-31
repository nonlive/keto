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
