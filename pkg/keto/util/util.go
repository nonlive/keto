package util

import (
	"fmt"
	"sort"
	"strings"

	"github.com/UKHomeOffice/keto/pkg/model"
)

// LabelsToKVs returns a string of labels in k=v,k=v format as a string.
func LabelsToKVs(m model.Labels) string {
	s := []string{}

	for k, v := range m {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

// KVsToLabels turns a list of k=v pairs into model.Labels.
func KVsToLabels(kvs []string) model.Labels {
	labels := model.Labels{}
	for _, kv := range kvs {
		s := strings.SplitN(kv, "=", 2)
		if len(s) == 2 {
			labels[s[0]] = s[1]
		}
	}
	return labels
}
