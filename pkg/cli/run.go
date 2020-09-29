package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/controller"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
	"github.com/suzuki-shunsuke/buildflow/pkg/file"
	"github.com/suzuki-shunsuke/buildflow/pkg/github"
	"github.com/suzuki-shunsuke/go-findconfig/findconfig"
	"github.com/urfave/cli/v2"
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

func (runner Runner) action(c *cli.Context) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	reader := config.Reader{
		ExistFile: findconfig.Exist,
	}
	cfg, err := reader.FindAndRead(c.String("config"), wd)
	if err != nil {
		return err
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
		"when":      cfg.When,
		"owner":     cfg.Owner,
		"repo":      cfg.Repo,
		"log_level": cfg.LogLevel,
	}).Debug("config")
	ex, err := expr.NewBool(cfg.When)
	if err != nil {
		return fmt.Errorf("it is failed to compile the expression. Please check the expression: %w", err)
	}

	ctrl := controller.Controller{
		Config:     cfg,
		GitHub:     ghClient,
		Expr:       ex,
		Executor:   execute.New(),
		Stdout:     os.Stdout,
		Stderr:     os.Stdout,
		Timer:      timer{},
		FileReader: file.Reader{},
	}

	return ctrl.Run(c.Context)
}

type timer struct{}

func (t timer) Now() time.Time {
	return time.Now()
}
