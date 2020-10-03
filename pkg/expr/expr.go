package expr

import (
	"context"
	"errors"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
)

type Program struct {
	script *tengo.Script
}

func New(expression string) (Program, error) {
	if expression == "" {
		return Program{}, nil
	}

	script := tengo.NewScript([]byte(expression))
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	return Program{
		script: script,
	}, nil
}

func (prog Program) Run(params map[string]interface{}) (interface{}, error) {
	if prog.script == nil {
		return nil, nil
	}
	for k, v := range params {
		if err := prog.script.Add(k, v); err != nil {
			return nil, err
		}
	}
	compiled, err := prog.script.RunContext(context.Background())
	if err != nil {
		return nil, err
	}

	if !compiled.IsDefined(constant.Result) {
		return nil, constant.ErrNoBoolVariable
	}
	v := compiled.Get(constant.Result)
	return v.Value(), nil
}

type BoolProgram struct {
	script *tengo.Script
}

func NewBool(expression string) (BoolProgram, error) {
	if expression == "" {
		return BoolProgram{}, nil
	}

	script := tengo.NewScript([]byte(expression))
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	return BoolProgram{
		script: script,
	}, nil
}

func (prog BoolProgram) Match(params map[string]interface{}) (bool, error) {
	if prog.script == nil {
		return true, nil
	}

	for k, v := range params {
		if err := prog.script.Add(k, v); err != nil {
			return false, err
		}
	}
	compiled, err := prog.script.RunContext(context.Background())
	if err != nil {
		return false, err
	}
	if !compiled.IsDefined(constant.Result) {
		return false, constant.ErrNoBoolVariable
	}
	v := compiled.Get(constant.Result)
	if t := v.ValueType(); t != "bool" {
		return false, errors.New(`the type of the variable "result" should be bool, but actually ` + t)
	}
	return v.Bool(), nil
}
