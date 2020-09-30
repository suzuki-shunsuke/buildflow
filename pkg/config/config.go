package config

import (
	"os"

	"github.com/suzuki-shunsuke/go-ci-env/cienv"
)

const (
	taskTypeCommand = "command"
)

type Phase struct {
	Name      string
	Tasks     []Task
	Condition PhaseCondition
}

type PhaseCondition struct {
	Skip Bool
	Exit Bool
	Fail Bool
}

type Config struct {
	Phases      []Phase
	Owner       string
	Repo        string
	When        Bool
	LogLevel    string `yaml:"log_level"`
	GitHubToken string `yaml:"github_token"`
	Env         Env    `yaml:"-"`
	Parallelism int
	PR          bool
}

type Env struct {
	Owner        string
	Repo         string
	PRNumber     int
	Branch       string
	SHA          string
	Tag          string
	Ref          string
	PRBaseBranch string
	IsPR         bool
	CI           bool
}

func Set(cfg Config) (Config, error) {
	cfg = setDefault(setEnv(cfg))
	for i, phase := range cfg.Phases {
		for j, task := range phase.Tasks {
			if err := task.Set(); err != nil {
				return cfg, err
			}
			phase.Tasks[j] = task
		}
		cfg.Phases[i] = phase
	}
	return cfg, nil
}

func setDefault(cfg Config) Config {
	cfg.When.SetDefaultBool(true)

	for i, phase := range cfg.Phases {
		phase.Condition.Skip.SetDefaultBool(false)
		phase.Condition.Exit.SetDefaultBool(false)
		phase.Condition.Fail.SetDefaultBool(false)

		for j, task := range phase.Tasks {
			if task.Command.Command != "" {
				task.Command = task.Command.SetDefault()
			}
			task.When.SetDefaultBool(true)
			phase.Tasks[j] = task
		}
		cfg.Phases[i] = phase
	}
	return cfg
}

func setEnv(cfg Config) Config {
	if cfg.GitHubToken == "" {
		cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	}
	if cfg.GitHubToken == "" {
		cfg.GitHubToken = os.Getenv("GITHUB_ACCESS_TOKEN")
	}
	platform := cienv.Get()
	cfg.Env.Owner = cfg.Owner
	cfg.Env.Repo = cfg.Repo
	if platform != nil {
		cfg.Env.CI = true
		if pr, err := platform.PRNumber(); err == nil {
			cfg.Env.PRNumber = pr
		}
		if cfg.Owner == "" {
			cfg.Owner = platform.RepoOwner()
			cfg.Env.Owner = cfg.Owner
		}
		if cfg.Repo == "" {
			cfg.Repo = platform.RepoName()
			cfg.Env.Repo = cfg.Repo
		}
		cfg.Env.Branch = platform.Branch()
		cfg.Env.SHA = platform.SHA()
		cfg.Env.Tag = platform.Tag()
		cfg.Env.Ref = platform.Ref()
		cfg.Env.PRBaseBranch = platform.PRBaseBranch()
		cfg.Env.IsPR = platform.IsPR()
	}
	return cfg
}
