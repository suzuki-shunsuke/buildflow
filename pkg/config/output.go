package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Script struct {
	Prog expr.Program
}

func (script Script) Run(params map[string]interface{}) (interface{}, error) {
	return script.Prog.Run(params)
}

func (script *Script) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := ""
	if err := unmarshal(&val); err != nil {
		return err
	}
	prog, err := expr.New(val)
	if err != nil {
		return err
	}
	script.Prog = prog
	return nil
}
