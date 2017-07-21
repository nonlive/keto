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

	"github.com/UKHomeOffice/keto/pkg/keto/util"
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
	clusterColumns  = []string{"NAME", "LABELS"}
	nodePoolColumns = []string{"NAME", "CLUSTER", "KUBEVERSION", "OSVERSION", "MACHINETYPE", "LABELS"}
)

// GetPrinter configures a new tabwriter Writer and returns it.
func GetPrinter(out io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(out, tabwriterMinWidth, tabwriterWidth, tabwriterPadding, tabwriterPadChar, tabwriterFlags)
}

// PrintClusters formats a slice of clusters into [][]string format with
// optional headers and calls writeToPrinter to write to w.
func PrintClusters(w *tabwriter.Writer, clusters []*model.Cluster, headers bool) error {
	data := [][]string{}
	if headers {
		data = append(data, clusterColumns)
	}
	for _, c := range clusters {
		labels := util.StringMapToKVs(c.Labels)
		data = append(data, []string{c.Name, labels})
	}
	fmt.Fprintln(w, formatData(data))
	return w.Flush()
}

// PrintMasterPool formats a slice of master pools into [][]string format with
// optional headers and calls writeToPrinter to write to w.
func PrintMasterPool(w *tabwriter.Writer, pools []*model.MasterPool, headers bool) error {
	data := [][]string{}
	if headers {
		data = append(data, nodePoolColumns)
	}
	for _, p := range pools {
		labels := util.StringMapToKVs(p.Labels)
		data = append(data, []string{p.Name, p.ClusterName, p.KubeVersion, p.CoreOSVersion, p.MachineType, labels})
	}
	fmt.Fprintln(w, formatData(data))
	return w.Flush()
}

// PrintComputePool formats a slice of compute pools into [][]string format with
// optional headers and calls writeToPrinter to write to w.
func PrintComputePool(w *tabwriter.Writer, pools []*model.ComputePool, headers bool) error {
	data := [][]string{}
	if headers {
		data = append(data, nodePoolColumns)
	}
	for _, p := range pools {
		labels := util.StringMapToKVs(p.Labels)
		data = append(data, []string{p.Name, p.ClusterName, p.KubeVersion, p.CoreOSVersion, p.MachineType, labels})
	}
	fmt.Fprintln(w, formatData(data))
	return w.Flush()
}

// formatData formats data of slices of string slices ready for tabwriter.
func formatData(data [][]string) string {
	rows := []string{}
	for _, v := range data {
		rows = append(rows, strings.Join(v, "\t"))
	}
	return strings.Join(rows, "\n")
}
