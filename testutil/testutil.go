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

// Package testutil provides test utility functions.
package testutil

import (
	"testing"

	"strings"
)

// CheckTemplate is a testing helper function that attempts to find match
// string in s string.
func CheckTemplate(t *testing.T, s, match string) {
	if !strings.Contains(s, match) {
		t.Errorf("failed to render the template; %q not found", match)
	}
}
