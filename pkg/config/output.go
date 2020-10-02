package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Output struct {
	Name string
	Prog expr.Program
}

func (output Output) Run(params interface{}) (interface{}, error) {
	return output.Prog.Run(params)
}

func (output *Output) UnmarshalYAML(unmarshal func(interface{}) error) error {
	val := struct {
		Name  string
		Value string
	}{}
	if err := unmarshal(&val); err != nil {
		return err
	}
	output.Name = val.Name
	prog, err := expr.New(val.Value)
	if err != nil {
		return err
	}
	output.Prog = prog
	return nil
}
