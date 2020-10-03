package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Output struct {
	Prog expr.Program
}

func (output Output) Run(params map[string]interface{}) (interface{}, error) {
	return output.Prog.Run(params)
}

func (output *Output) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := ""
	if err := unmarshal(&val); err != nil {
		return err
	}
	prog, err := expr.New(val)
	if err != nil {
		return err
	}
	output.Prog = prog
	return nil
}
