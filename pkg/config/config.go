package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
	"github.com/suzuki-shunsuke/go-ci-env/cienv"
	"github.com/suzuki-shunsuke/go-convmap/convmap"
)

type Phase struct {
	Name      string
	Tasks     []Task
	Condition PhaseCondition
	Meta      map[string]interface{}
	Import    string
}

type PhaseCondition struct {
	Skip Bool
	Exit Bool
	Fail Bool
}

type BuildCondition struct {
	Skip Bool
	Fail Bool
}

type Config struct {
	Phases      []Phase
	Owner       string
	Repo        string
	Condition   BuildCondition
	LogLevel    string `yaml:"log_level"`
	GitHubToken string `yaml:"github_token"`
	Env         Env    `yaml:"-"`
	Parallelism int
	PR          bool
	Meta        map[string]interface{}
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

func convertMeta(meta map[string]interface{}) error {
	for k, a := range meta {
		b, err := convmap.Convert(a)
		if err != nil {
			return fmt.Errorf("parse meta: %w", err)
		}
		meta[k] = b
	}
	return nil
}

func Set(cfg Config) (Config, error) {
	cfg = setDefault(setEnv(cfg))
	if err := convertMeta(cfg.Meta); err != nil {
		return cfg, fmt.Errorf(".meta is invalid: %w", err)
	}
	phaseNames := make(map[string]struct{}, len(cfg.Phases))
	for i, phase := range cfg.Phases {
		if _, ok := phaseNames[phase.Name]; ok {
			return cfg, errors.New("phase name is duplicated: " + phase.Name)
		}
		phaseNames[phase.Name] = struct{}{}
		if err := convertMeta(phase.Meta); err != nil {
			return cfg, fmt.Errorf("phase is invalid: %s: %w", phase.Name, err)
		}
		for j, task := range phase.Tasks {
			if err := task.Set(); err != nil {
				return cfg, fmt.Errorf("task is invalid: %w", err)
			}
			phase.Tasks[j] = task
		}
		cfg.Phases[i] = phase
	}
	return cfg, nil
}

func setDefault(cfg Config) Config {
	if !cfg.Condition.Fail.Initialized {
		b, err := expr.NewBool(`
result := false
for phase in Phases {
  if phase.Status == "failed" {
    result = true
	}
}`)
		if err != nil {
			panic(err)
		}
		cfg.Condition.Fail.Initialized = true
		cfg.Condition.Fail.Prog = b
		cfg.Condition.Fail.Fixed = false
	}
	cfg.Condition.Skip.SetDefaultBool(false)

	for i, phase := range cfg.Phases {
		phase.Condition.Skip.SetDefaultBool(false)
		phase.Condition.Exit.SetDefaultBool(false)
		phase.Condition.Fail.SetDefaultBool(false)

		if !phase.Condition.Fail.Initialized {
			b, err := expr.NewBool(`
result := false
for task in Tasks {
  if task.Status == "failed" {
    result = true
	}
}`)
			if err != nil {
				panic(err)
			}
			phase.Condition.Fail.Initialized = true
			phase.Condition.Fail.Prog = b
			phase.Condition.Fail.Fixed = false
		}

		for j, task := range phase.Tasks {
			if task.Command.Command.Text != "" || task.Command.CommandFile != "" {
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
