package config

import "github.com/suzuki-shunsuke/buildflow/pkg/expr"

type Items struct {
	Items   interface{}
	Program expr.Program
}

type Item struct {
	Key   interface{}
	Value interface{}
}

func (items Items) Run(params map[string]interface{}) (interface{}, error) {
	a, err := items.Program.Run(params)
	if err != nil {
		return nil, err
	}
	return a, nil
}
