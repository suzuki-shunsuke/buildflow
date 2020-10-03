package expr

import (
	"context"
	"errors"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
)

type Program struct {
	source string
}

func New(expression string) (Program, error) {
	return Program{
		source: expression,
	}, nil
}

func (prog Program) Run(params map[string]interface{}) (interface{}, error) {
	if prog.source == "" {
		return nil, nil
	}
	script := tengo.NewScript([]byte(prog.source))
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))
	for k, v := range params {
		if err := script.Add(k, v); err != nil {
			return nil, err
		}
	}
	compiled, err := script.RunContext(context.Background())
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
	source string
}

func NewBool(expression string) (BoolProgram, error) {
	return BoolProgram{
		source: expression,
	}, nil
}

func (prog BoolProgram) Match(params map[string]interface{}) (bool, error) {
	if prog.source == "" {
		return true, nil
	}
	script := tengo.NewScript([]byte(prog.source))
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	for k, v := range params {
		if err := script.Add(k, v); err != nil {
			return false, err
		}
	}
	compiled, err := script.RunContext(context.Background())
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
