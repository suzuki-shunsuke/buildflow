package config

type Command struct {
	Shell     string
	ShellOpts []string `yaml:"shell_options"`
	Command   Template
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
