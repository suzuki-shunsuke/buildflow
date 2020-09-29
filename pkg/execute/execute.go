package execute

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Songmu/timeout"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/go-error-with-exit-code/ecerror"
)

type Executor struct {
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Environ []string
}

func New() Executor {
	return Executor{
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Environ: os.Environ(),
	}
}

type Params struct {
	Cmd        string
	Args       []string
	WorkingDir string
	Envs       []string
	Quiet      bool
	DryRun     bool
	Timeout    Timeout
	Stdout     io.Writer
	Stderr     io.Writer
	TaskName   string
}

type Timeout struct {
	Duration  time.Duration
	KillAfter time.Duration `yaml:"kill_after"`
}

func (exc Executor) Run(ctx context.Context, params Params) (domain.CommandResult, error) {
	cmd := exec.Command(params.Cmd, params.Args...) //nolint:gosec
	bufStdout := &bytes.Buffer{}
	bufStderr := &bytes.Buffer{}
	combinedOutput := &bytes.Buffer{}
	if params.Stdout == nil {
		params.Stdout = exc.Stdout
	}
	if params.Stderr == nil {
		params.Stderr = exc.Stderr
	}
	cmd.Stdout = io.MultiWriter(params.Stdout, bufStdout, combinedOutput)
	cmd.Stderr = io.MultiWriter(params.Stderr, bufStderr, combinedOutput)
	cmd.Stdin = exc.Stdin
	cmd.Dir = params.WorkingDir

	cmd.Env = append(exc.Environ, params.Envs...) //nolint:gocritic
	if !params.Quiet {
		fmt.Fprintln(cmd.Stderr, "+ "+params.Cmd+" "+strings.Join(params.Args, " "))
	}
	if params.DryRun {
		return domain.CommandResult{}, nil
	}
	tio := timeout.Timeout{
		Cmd:       cmd,
		Duration:  params.Timeout.Duration,
		KillAfter: params.Timeout.KillAfter,
	}
	exitStatus, err := tio.RunContext(ctx)
	result := domain.CommandResult{
		Cmd:            cmd.String(),
		Stdout:         bufStdout.String(),
		Stderr:         bufStderr.String(),
		CombinedOutput: combinedOutput.String(),
	}
	if err != nil {
		result.ExitCode = -1
		return result, err
	}
	result.ExitCode = exitStatus.Code
	if exitStatus.Code != 0 {
		return result, ecerror.Wrap(err, exitStatus.Code)
	}
	return result, nil
}
