package controller

import (
	"context"
	"io"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	gh "github.com/suzuki-shunsuke/buildflow/pkg/github"
)

type Controller struct {
	GitHub     GitHub
	Config     config.Config
	Executor   Executor
	FileReader FileReader
	Timer      Timer
	Stdout     io.Writer
	Stderr     io.Writer
}

type Executor interface {
	Run(ctx context.Context, params execute.Params) (domain.CommandResult, error)
}

type Timer interface {
	Now() time.Time
}

type FileReader interface {
	Read(path string) (domain.FileResult, error)
}

type GitHub interface {
	GetPR(ctx context.Context, params gh.ParamsGetPR) (*github.PullRequest, *github.Response, error)
	GetPRFiles(ctx context.Context, params gh.ParamsGetPRFiles) ([]*github.CommitFile, *github.Response, error)
	ListPRsWithCommit(ctx context.Context, params gh.ParamsListPRsWithCommit) ([]*github.PullRequest, *github.Response, error)
}

type Expr interface {
	Match(params interface{}) (bool, error)
}
