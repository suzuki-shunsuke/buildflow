package config

import (
	"errors"

	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
)

type Task struct {
	Name          Template
	Type          string `yaml:"-"`
	When          Bool
	Dependency    Dependency
	Command       Command
	ReadFile      ReadFile  `yaml:"read_file"`
	WriteFile     WriteFile `yaml:"write_file"`
	HTTP          HTTP
	Timeout       execute.Timeout
	Items         interface{}
	Item          Item `yaml:"-"`
	CompiledItems Items
	Meta          map[string]interface{}
	Output        Script
	Input         Script
}

type WriteFile struct {
	Path     Template
	Template Template
}

func (task *Task) Set() error {
	if err := task.SetType(); err != nil {
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

func (task *Task) SetType() error {
	if task.Command.Command.Text != "" {
		task.Type = constant.Command
		return nil
	}
	if task.ReadFile.Path.Text != "" {
		task.Type = constant.ReadFile
		return nil
	}
	if task.WriteFile.Path.Text != "" {
		task.Type = constant.WriteFile
		return nil
	}
	if task.HTTP.URL != "" {
		task.Type = constant.HTTP
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
