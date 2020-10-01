package config

import (
	"fmt"

	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Bool struct {
	Prog        expr.BoolProgram
	Initialized bool
	Fixed       bool
	FixedValue  bool
}

func (b Bool) Match(params interface{}) (bool, error) {
	if b.Fixed {
		return b.FixedValue, nil
	}
	return b.Prog.Match(params)
}

func (b *Bool) SetBool(f bool) {
	b.FixedValue = f
	b.Fixed = true
}

func (b *Bool) SetDefaultBool(f bool) {
	if !b.Initialized {
		b.SetBool(f)
	}
}

func (b *Bool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var val interface{}
	if err := unmarshal(&val); err != nil {
		return err
	}
	if val == nil {
		b.Fixed = true
		b.FixedValue = false
		return nil
	}
	switch a := val.(type) {
	case string:
		prog, err := expr.NewBool(a)
		if err != nil {
			return err
		}
		b.Prog = prog
		b.Fixed = false
		b.Initialized = true
		return nil
	case bool:
		b.FixedValue = a
		b.Fixed = true
		b.Initialized = true
		return nil
	default:
		return fmt.Errorf("the value should be true or false or string: %v", val)
	}
}
