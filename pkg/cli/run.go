package cli

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/controller"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/file"
	"github.com/suzuki-shunsuke/buildflow/pkg/github"
	"github.com/suzuki-shunsuke/go-findconfig/findconfig"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func (runner Runner) setCLIArg(c *cli.Context, cfg config.Config) config.Config {
	if owner := c.String("owner"); owner != "" {
		cfg.Owner = owner
	}
	if repo := c.String("repo"); repo != "" {
		cfg.Repo = repo
	}
	if token := c.String("github-token"); token != "" {
		cfg.GitHubToken = token
	}
	if logLevel := c.String("log-level"); logLevel != "" {
		cfg.LogLevel = logLevel
	}
	return cfg
}

func (runner Runner) importPhaseConfig(cfgPhases []config.Phase, wd string) ([]config.Phase, error) { //nolint:dupl
	phases := []config.Phase{}
	for _, phase := range cfgPhases {
		if phase.Import == "" {
			phases = append(phases, phase)
			continue
		}
		p := phase.Import
		if !filepath.IsAbs(p) {
			p = filepath.Join(wd, p)
		}
		arr, err := func() ([]config.Phase, error) {
			arr := []config.Phase{}
			file, err := os.Open(p)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			if err := yaml.NewDecoder(file).Decode(&arr); err != nil {
				return nil, err
			}
			return arr, nil
		}()
		if err != nil {
			return phases, err
		}
		phases = append(phases, arr...)
	}
	return phases, nil
}

func (runner Runner) importTaskConfig(cfgTasks []config.Task, wd string) ([]config.Task, error) { //nolint:dupl
	tasks := []config.Task{}
	for _, task := range cfgTasks {
		if task.Import == "" {
			tasks = append(tasks, task)
			continue
		}
		p := task.Import
		if !filepath.IsAbs(p) {
			p = filepath.Join(wd, p)
		}
		arr, err := func() ([]config.Task, error) {
			arr := []config.Task{}
			file, err := os.Open(p)
			if err != nil {
				return nil, err
			}
			defer file.Close()
			if err := yaml.NewDecoder(file).Decode(&arr); err != nil {
				return nil, err
			}
			return arr, nil
		}()
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, arr...)
	}
	return tasks, nil
}

func (runner Runner) importConfig(cfg config.Config, wd string) (config.Config, error) {
	phases, err := runner.importPhaseConfig(cfg.Phases, wd)
	if err != nil {
		return cfg, err
	}
	for i, phase := range phases {
		tasks, err := runner.importTaskConfig(phase.Tasks, wd)
		if err != nil {
			return cfg, err
		}
		phase.Tasks = tasks
		phases[i] = phase
	}
	cfg.Phases = phases
	return cfg, nil
}

func (runner Runner) action(c *cli.Context) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	reader := config.Reader{
		ExistFile: findconfig.Exist,
	}
	cfg, cfgPath, err := reader.FindAndRead(c.String("config"), wd)
	if err != nil {
		return err
	}

	if c, err := runner.importConfig(cfg, wd); err != nil {
		return err
	} else {
		cfg = c
	}

	cfg = runner.setCLIArg(c, cfg)
	cfg, err = config.Set(cfg)
	if err != nil {
		return err
	}

	ghClient := github.New(c.Context, github.ParamsNew{
		Token: cfg.GitHubToken,
	})

	if cfg.LogLevel != "" {
		lvl, err := logrus.ParseLevel(cfg.LogLevel)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"log_level": cfg.LogLevel,
			}).WithError(err).Error("the log level is invalid")
		}
		logrus.SetLevel(lvl)
	}

	logrus.WithFields(logrus.Fields{
		"owner":     cfg.Owner,
		"repo":      cfg.Repo,
		"log_level": cfg.LogLevel,
	}).Debug("config")

	ctrl := controller.Controller{
		Config:     cfg,
		GitHub:     ghClient,
		Executor:   execute.New(),
		Stdout:     os.Stdout,
		Stderr:     os.Stdout,
		Timer:      timer{},
		FileReader: file.Reader{},
		FileWriter: file.Writer{},
	}

	return ctrl.Run(c.Context, filepath.Dir(cfgPath))
}

type timer struct{}

func (t timer) Now() time.Time {
	return time.Now()
}
