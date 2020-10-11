package config

import (
	"fmt"

	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Dependency struct {
	Names   []string
	Program expr.BoolProgram
}

func (dependency *Dependency) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var val interface{}
	if err := unmarshal(&val); err != nil {
		return err
	}
	if val == nil {
		return nil
	}
	switch a := val.(type) {
	case string:
		prog, err := expr.NewBool(a)
		if err != nil {
			return err
		}
		dependency.Program = prog
		return nil
	case []string:
		dependency.Names = a
		return nil
	case []interface{}:
		names := make([]string, len(a))
		for i, name := range a {
			s, ok := name.(string)
			if !ok {
				return fmt.Errorf("the value should be a string, which is a dependency name: %v", val)
			}
			names[i] = s
		}
		dependency.Names = names
		return nil
	default:
		return fmt.Errorf("the value should be a tengo script or a list of dependencies: %v", val)
	}
}
