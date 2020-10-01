package controller

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
)

type Task struct {
	Result     domain.Result
	Config     config.Task
	Executor   Executor
	FileReader FileReader
	Timer      Timer
	Stdout     io.Writer
	Stderr     io.Writer
}

func (task Task) runCommand(ctx context.Context) (domain.CommandResult, error) {
	if task.Config.Timeout.Duration == 0 {
		task.Config.Timeout.Duration = 1 * time.Hour
	}
	result, err := task.Executor.Run(ctx, execute.Params{
		Cmd:      task.Config.Command.Shell,
		Args:     append(task.Config.Command.ShellOpts, task.Config.Command.Command.Text),
		Timeout:  task.Config.Timeout,
		TaskName: task.Config.Name.Text,
		Stdout:   task.Stdout,
		Stderr:   task.Stderr,
		Envs:     task.Config.Command.Env.Compiled,
	})
	return result, err
}

func (task Task) Run(ctx context.Context) (domain.Result, error) {
	startTime := task.Timer.Now()
	switch task.Config.Type {
	case "command":
		cmdResult, err := task.runCommand(ctx)
		return domain.Result{
			Command: cmdResult,
			Time: domain.Time{
				Start: startTime,
				End:   task.Timer.Now(),
			},
		}, err
	case "file":
		fileResult, err := task.FileReader.Read(task.Config.ReadFile.Path.Text)
		return domain.Result{
			File: fileResult,
			Time: domain.Time{
				Start: startTime,
				End:   task.Timer.Now(),
			},
		}, err
	}
	return domain.Result{}, errors.New("invalid task type: " + task.Config.Type + ", task name: " + task.Config.Name.Text)
}
