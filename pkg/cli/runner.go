package cli

import (
	"context"
	"errors"
	"io"

	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
	"github.com/urfave/cli/v2"
)

type Runner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (runner Runner) Run(ctx context.Context, args ...string) error {
	app := cli.App{
		Name:    "buildflow",
		Usage:   "run build. https://github.com/suzuki-shunsuke/buildflow",
		Version: constant.Version,
		Commands: []*cli.Command{
			{
				Name:   "run",
				Usage:  "run build",
				Action: runner.action,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "owner",
						Usage: "repository owner",
					},
					&cli.StringFlag{
						Name:  "repo",
						Usage: "repository name",
					},
					&cli.StringFlag{
						Name:  "github-token",
						Usage: "GitHub Access Token [$GITHUB_TOKEN, $GITHUB_ACCESS_TOKEN]",
					},
					&cli.StringFlag{
						Name:  "log-level",
						Usage: "log level",
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "configuration file path",
					},
				},
			},
			{
				Name:   "init",
				Usage:  "generate a configuration file if it doesn't exist",
				Action: runner.initAction,
			},
		},
	}

	return app.RunContext(ctx, args)
}

var (
	ErrGitHubAccessTokenIsRequired error = errors.New("GitHub Access Token is required")
	ErrOwnerIsRequired             error = errors.New("owner is required")
	ErrRepoIsRequired              error = errors.New("repo is required")
)
