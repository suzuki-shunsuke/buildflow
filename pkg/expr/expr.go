package expr

import (
	"context"
	"errors"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
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

func (prog Program) Run(params map[string]interface{}) (map[string]interface{}, error) {
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
	vars := compiled.GetAll()
	m := make(map[string]interface{}, len(vars))
	for _, v := range vars {
		m[v.Name()] = v.Value()
	}

	return m, nil
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

var ErrNoBoolVariable = errors.New(`the variable "answer" isn't defined`)

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
	if !compiled.IsDefined("answer") {
		return false, ErrNoBoolVariable
	}
	v := compiled.Get("answer")
	if t := v.ValueType(); t != "bool" {
		return false, errors.New(`the type of the variable "answer" should be bool, but actually ` + t)
	}
	return v.Bool(), nil
}
