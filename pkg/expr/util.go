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

func GetUtil() map[string]interface{} {
	return map[string]interface{}{
		"labelNames": LabelNames,
		"env":        os.Getenv,
		"string": map[string]interface{}{
			"split":     strings.Split,
			"trimSpace": strings.TrimSpace,
		},
	}
}
