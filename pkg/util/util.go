package util

import (
	"os"
	"strings"
	"text/template"
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

func getTaskByName(tasks []map[string]interface{}, name string) map[string]interface{} {
	for _, task := range tasks {
		if n, ok := task["Name"]; ok && n == name {
			return task
		}
	}
	return nil
}

func GetTemplateUtil() template.FuncMap {
	return template.FuncMap{
		"LabelNames":    LabelNames,
		"GetTaskByName": getTaskByName,
	}
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
		"GetTaskByName": getTaskByName,
	}
}
