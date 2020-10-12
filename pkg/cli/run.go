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

func (runner Runner) importConfig(cfg config.Config, wd string) (config.Config, error) {
	phases := []config.Phase{}
	for _, phase := range cfg.Phases {
		if phase.Import == "" {
			phases = append(phases, phase)
			continue
		}
		p := phase.Import
		if !filepath.IsAbs(p) {
			p = filepath.Join(wd, p)
		}
		arr := []config.Phase{}
		file, err := os.Open(p)
		if err != nil {
			return cfg, err
		}
		defer file.Close()
		if err := yaml.NewDecoder(file).Decode(&arr); err != nil {
			return cfg, err
		}
		phases = append(phases, arr...)
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
