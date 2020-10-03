package util

import (
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
