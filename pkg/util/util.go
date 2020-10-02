package util

import (
	"os"
	"strings"
	"text/template"
)

func labelNames(labels interface{}) []string {
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

func getTaskByName(tasks []interface{}, name string) map[string]interface{} {
	for _, task := range tasks {
		m := task.(map[string]interface{})
		if n, ok := m["Name"]; ok && n == name {
			return m
		}
	}
	return nil
}

func GetTemplateUtil() template.FuncMap {
	return template.FuncMap{
		"LabelNames":    labelNames,
		"GetTaskByName": getTaskByName,
	}
}

func GetUtil() map[string]interface{} {
	return map[string]interface{}{
		"LabelNames": labelNames,
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
