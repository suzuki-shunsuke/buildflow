package config

import (
	"errors"

	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Task struct {
	Name               Template
	Type               string `yaml:"-"`
	When               Bool
	Dependency         interface{}
	CompiledDependency Dependency `yaml:"-"`
	Command            Command
	ReadFile           ReadFile `yaml:"read_file"`
	HTTP               HTTP
	Timeout            execute.Timeout
	Items              interface{}
	Item               Item `yaml:"-"`
	CompiledItems      Items
	Meta               map[string]interface{}
	Output             Output
}

type Dependency struct {
	Names   []string
	Program expr.BoolProgram
}

type Items struct {
	Items   interface{}
	Program expr.Program
}

type Item struct {
	Key   interface{}
	Value interface{}
}

func (task *Task) Set() error {
	if err := task.SetType(); err != nil {
		return err
	}

	if err := task.CompileDependency(); err != nil {
		return err
	}

	if s, ok := task.Items.(string); ok {
		prog, err := expr.New(s)
		if err != nil {
			return err
		}
		task.CompiledItems = Items{
			Program: prog,
		}
	}

	return nil
}

func (task *Task) CompileDependency() error {
	if task.Dependency == nil {
		return nil
	}
	if s, ok := task.Dependency.(string); ok {
		prog, err := expr.NewBool(s)
		if err != nil {
			return err
		}
		task.CompiledDependency.Program = prog
		return nil
	}
	if names, ok := task.Dependency.([]interface{}); ok {
		ns := make([]string, len(names))
		for i, n := range names {
			name, ok := n.(string)
			if !ok {
				return errors.New("dependency should be either string or []string")
			}
			ns[i] = name
		}
		task.CompiledDependency.Names = ns
		return nil
	}
	return errors.New("dependency should be either string or []string")
}

func (task *Task) SetType() error {
	if task.Command.Command.Text != "" {
		task.Type = "command"
		return nil
	}
	if task.ReadFile.Path.Text != "" {
		task.Type = "file"
		return nil
	}
	if task.HTTP.URL != "" {
		task.Type = "http"
		return nil
	}
	return errors.New("task must be either command, file, and http")
}

type ReadFile struct {
	Path   Template
	Format string
}

type HTTP struct {
	URL string
}
