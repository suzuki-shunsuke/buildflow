package controller

import (
	"errors"
	"reflect"

	"github.com/suzuki-shunsuke/buildflow/pkg/config"
)

func expandSlice(task config.Task, value reflect.Value, params Params) ([]config.Task, error) {
	size := value.Len()
	tasks := make([]config.Task, size)
	for i := 0; i < size; i++ {
		val := value.Index(i).Interface()
		params.Item = config.Item{
			Key:   i,
			Value: val,
		}
		name, err := task.Name.New(params)
		if err != nil {
			return nil, err
		}
		t := task
		t.Name = name
		t.Item = config.Item{
			Key:   i,
			Value: val,
		}
		if err != nil {
			return nil, err
		}
		tasks[i] = t
	}
	return tasks, nil
}

func expandMap(task config.Task, value reflect.Value, params Params) ([]config.Task, error) {
	size := value.Len()
	tasks := make([]config.Task, 0, size)
	for _, key := range value.MapKeys() {
		k := key.Interface()
		val := value.MapIndex(key).Interface()
		params.Item = config.Item{
			Key:   k,
			Value: val,
		}

		name, err := task.Name.New(params)
		if err != nil {
			return nil, err
		}
		t := task
		t.Name = name
		t.Item = config.Item{
			Key:   k,
			Value: val,
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func expandItems(task config.Task, items interface{}, templateParams Params) ([]config.Task, error) {
	value := reflect.ValueOf(items)
	switch value.Kind() { //nolint:exhaustive
	case reflect.Slice:
		return expandSlice(task, value, templateParams)
	case reflect.Map:
		return expandMap(task, value, templateParams)
	default:
		return nil, errors.New("task.items should be either expression string or array. invalid kind: " + value.Kind().String())
	}
}

func Expand(task config.Task, params Params) ([]config.Task, error) {
	if task.Items == nil {
		name, err := task.Name.New(params)
		if err != nil {
			return nil, err
		}
		t := task
		t.Name = name
		return []config.Task{t}, nil
	}
	if _, ok := task.Items.(string); ok {
		items, err := task.CompiledItems.Program.Run(params)
		if err != nil {
			return nil, err
		}
		return expandItems(task, items, params)
	}
	return expandItems(task, task.Items, params)
}
