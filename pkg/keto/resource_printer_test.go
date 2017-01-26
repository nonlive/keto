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

package keto

import (
	"strings"
	"testing"
)

func TestFormatData(t *testing.T) {
	testCases := []struct {
		name  string
		input [][]string
		want  string
	}{
		{
			"no data",
			[][]string{},
			"",
		},
		{
			"nil input",
			nil,
			"",
		},
		{
			"must be joined with tabs",
			[][]string{
				nodePoolColumns,
			},
			strings.Join(nodePoolColumns, "\t"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatData(tc.input); got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
