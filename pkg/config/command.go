package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/template"
)

type Command struct {
	Shell       string
	ShellOpts   []string `yaml:"shell_options"`
	Command     Template
	CommandFile string `yaml:"command_file"`
	Env         Envs
}

func (cmd Command) SetDefault() Command {
	if cmd.Shell == "" {
		cmd.Shell = "/bin/sh"
		if cmd.ShellOpts == nil {
			cmd.ShellOpts = []string{"-c"}
		}
	}
	return cmd
}

type Envs struct {
	Vars     []EnvVar
	Compiled []string
}

type EnvVar struct {
	Key       template.Template
	Value     template.Template
	ValueFile string `yaml:"value_file"`
}

func (envs *Envs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := []EnvVar{}
	if err := unmarshal(&m); err != nil {
		return err
	}
	envs.Vars = m
	return nil
}
