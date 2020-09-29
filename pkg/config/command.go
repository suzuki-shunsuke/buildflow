package config

import (
	"github.com/suzuki-shunsuke/buildflow/pkg/template"
)

type Command struct {
	Shell           string
	ShellOpts       []string `yaml:"shell_options"`
	Command         string
	CompiledCommand template.Template
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
