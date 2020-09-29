package config

import (
	"os"

	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
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
	Skip         string
	Exit         string
	Fail         string
	CompiledSkip expr.BoolProgram `yaml:"-"`
	CompiledExit expr.BoolProgram `yaml:"-"`
	CompiledFail expr.BoolProgram `yaml:"-"`
}

type Config struct {
	Phases      []Phase
	Owner       string
	Repo        string
	When        string
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

const (
	falseString = "false"
)

func Set(cfg Config) (Config, error) {
	cfg = setDefault(setEnv(cfg))
	for i, phase := range cfg.Phases {
		if phase.Condition.Skip == "" {
			phase.Condition.Skip = falseString
		}
		if phase.Condition.Exit == "" {
			phase.Condition.Exit = falseString
		}
		if phase.Condition.Fail == "" {
			phase.Condition.Fail = falseString
		}
		if e, err := expr.NewBool(phase.Condition.Skip); err != nil {
			return cfg, err
		} else { //nolint:golint
			phase.Condition.CompiledSkip = e
		}
		if e, err := expr.NewBool(phase.Condition.Exit); err != nil {
			return cfg, err
		} else { //nolint:golint
			phase.Condition.CompiledExit = e
		}
		if e, err := expr.NewBool(phase.Condition.Fail); err != nil {
			return cfg, err
		} else { //nolint:golint
			phase.Condition.CompiledFail = e
		}
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
	if cfg.When == "" {
		cfg.When = "true"
	}

	for i, phase := range cfg.Phases {
		for j, task := range phase.Tasks {
			if task.Command.Command != "" {
				task.Command = task.Command.SetDefault()
			}
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
