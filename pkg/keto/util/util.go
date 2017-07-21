package util

import (
	"fmt"
	"sort"
	"strings"
)

// StringMapToKVs returns a string of labels in k=v,k=v format as a string.
func StringMapToKVs(m map[string]string) string {
	s := []string{}

	for k, v := range m {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

// KVsToStringMap turns a list of k=v pairs into a string map of strings.
func KVsToStringMap(kvs []string) map[string]string {
	m := make(map[string]string)
	for _, kv := range kvs {
		s := strings.SplitN(kv, "=", 2)
		if len(s) == 2 {
			m[s[0]] = s[1]
		}
	}
	return m
}
