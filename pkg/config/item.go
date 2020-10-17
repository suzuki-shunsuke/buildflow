package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
	"github.com/suzuki-shunsuke/go-convmap/convmap"
)

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

func (items *Items) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var src interface{}
	if err := unmarshal(&src); err != nil {
		return err
	}
	switch t := src.(type) {
	case string:
		prog, err := expr.New(t)
		if err != nil {
			return err
		}
		items.Program = prog
		return nil
	default:
		if t == nil {
			return nil
		}
		a, err := convmap.Convert(t)
		if err != nil {
			return err
		}
		items.Items = a
		return nil
	}
}
