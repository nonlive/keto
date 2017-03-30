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
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/UKHomeOffice/keto/pkg/model"
)

const (
	tabwriterMinWidth = 10
	tabwriterWidth    = 4
	tabwriterPadding  = 3
	tabwriterPadChar  = ' '
	tabwriterFlags    = 0
)

var (
	clusterColumns  = []string{"NAME"}
	nodePoolColumns = []string{"NAME", "CLUSTER", "MACHINETYPE", "OSVERSION"}
)

// GetPrinter configures a new tabwriter Writer and returns it.
func GetPrinter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, tabwriterMinWidth, tabwriterWidth, tabwriterPadding, tabwriterPadChar, tabwriterFlags)
}

// PrintNodePool formats a slice of node pools into [][]string format with
// optional headers and calls writeToPrinter to write to w.
func PrintNodePool(w *tabwriter.Writer, pools []*model.NodePool, headers bool) error {
	// TODO
	data := [][]string{}
	if headers {
		data = append(data, nodePoolColumns)
	}
	for _, p := range pools {
		data = append(data, []string{p.Name, p.ClusterName, p.MachineType, p.OSVersion})
	}
	fmt.Fprintln(w, formatData(data))
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

// formatData formats data of slices of string slices ready for tabwriter.
func formatData(data [][]string) string {
	rows := []string{}
	for _, v := range data {
		rows = append(rows, strings.Join(v, "\t"))
	}
	return strings.Join(rows, "\n")
}
