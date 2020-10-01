package expr

import (
	"os"
	"strings"
)

func LabelNames(labels interface{}) []string {
	if labels == nil {
		return []string{}
	}
	a := labels.([]interface{})
	b := make([]string, len(a))
	for i, label := range a {
		b[i] = label.(map[string]interface{})["name"].(string)
	}
	return b
}

func listKeysOfMap(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func listValuesOfMap(m map[string]interface{}) []interface{} {
	vals := make([]interface{}, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}

func GetUtil() map[string]interface{} {
	return map[string]interface{}{
		"LabelNames": LabelNames,
		"Env":        os.Getenv,
		"String": map[string]interface{}{
			"Split":     strings.Split,
			"TrimSpace": strings.TrimSpace,
		},
		"Map": map[string]interface{}{
			"Keys":   listKeysOfMap,
			"Values": listValuesOfMap,
		},
	}
}
