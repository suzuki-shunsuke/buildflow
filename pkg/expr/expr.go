package expr

import (
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
)

type Program struct {
	prog *vm.Program
}

func New(expression string) (Program, error) {
	if expression == "" {
		return Program{}, nil
	}
	prog, err := expr.Compile(expression)
	if err != nil {
		return Program{}, err
	}
	return Program{
		prog: prog,
	}, nil
}

func (prog Program) Run(params interface{}) (interface{}, error) {
	if prog.prog == nil {
		return nil, nil
	}
	return expr.Run(prog.prog, params)
}

type BoolProgram struct {
	prog *vm.Program
}

func NewBool(expression string) (BoolProgram, error) {
	if expression == "" {
		return BoolProgram{}, nil
	}
	prog, err := expr.Compile(expression, expr.AsBool())
	if err != nil {
		return BoolProgram{}, err
	}
	return BoolProgram{
		prog: prog,
	}, nil
}

func (prog BoolProgram) Match(params interface{}) (bool, error) {
	if prog.prog == nil {
		return true, nil
	}
	output, err := expr.Run(prog.prog, params)
	if err != nil {
		return false, err
	}
	if f, ok := output.(bool); !ok || !f {
		return false, nil
	}
	return true, nil
}
