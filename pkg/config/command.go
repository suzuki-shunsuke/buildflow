package config

import "github.com/suzuki-shunsuke/buildflow/pkg/template"

type Command struct {
	Shell     string
	ShellOpts []string `yaml:"shell_options"`
	Command   Template
	Env       Envs
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
	Key   template.Template
	Value template.Template
}

func (envs *Envs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	m := map[string]string{}
	if err := unmarshal(&m); err != nil {
		return err
	}
	arr := make([]EnvVar, 0, len(m))
	for k, v := range m {
		key, err := template.Compile(k)
		if err != nil {
			return err
		}
		val, err := template.Compile(v)
		if err != nil {
			return err
		}
		arr = append(arr, EnvVar{
			Key:   key,
			Value: val,
		})
	}
	envs.Vars = arr
	return nil
}
