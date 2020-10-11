package controller

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
)

type Task struct {
	Result     domain.Result
	Config     config.Task
	Executor   Executor
	FileReader FileReader
	FileWriter FileWriter
	Timer      Timer
	Stdout     io.Writer
	Stderr     io.Writer
}

func (task Task) runCommand(ctx context.Context, wd string) (domain.CommandResult, error) {
	if task.Config.Timeout.Duration == 0 {
		task.Config.Timeout.Duration = 1 * time.Hour
	}
	result, err := task.Executor.Run(ctx, execute.Params{
		Cmd:        task.Config.Command.Shell,
		Args:       append(task.Config.Command.ShellOpts, task.Config.Command.Command.Text),
		Timeout:    task.Config.Timeout,
		TaskName:   task.Name(),
		Stdout:     task.Stdout,
		Stderr:     task.Stderr,
		WorkingDir: wd,
		Envs:       task.Config.Command.Env.Compiled,
	})
	return result, err
}

func (task Task) run(ctx context.Context, wd string) (domain.Result, error) {
	switch task.Config.Type {
	case constant.Command:
		cmdResult, err := task.runCommand(ctx, wd)
		return domain.Result{
			Command: cmdResult,
		}, err
	case constant.ReadFile:
		fileResult, err := task.FileReader.Read(task.Config.ReadFile.Path.Text)
		return domain.Result{
			File: fileResult,
		}, err
	case constant.WriteFile:
		// TODO append a new line
		fileResult, err := task.FileWriter.Write(
			task.Config.WriteFile.Path.Text, task.Config.WriteFile.Template.Text)
		return domain.Result{
			File: fileResult,
		}, err
	}
	return domain.Result{}, errors.New("invalid task type: " + task.Config.Type + ", task name: " + task.Name())
}

func (task Task) Run(ctx context.Context, wd string) (domain.Result, error) {
	startTime := task.Timer.Now()
	result, err := task.run(ctx, wd)
	result.Time.Start = startTime
	result.Time.End = task.Timer.Now()
	return result, err
}
