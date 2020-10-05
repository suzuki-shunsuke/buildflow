package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
	"github.com/suzuki-shunsuke/buildflow/pkg/locale"
)

type Phase struct {
	Config     config.Phase
	Tasks      []Task
	mutex      sync.RWMutex
	EventQueue EventQueue
	Stdout     io.Writer
	Stderr     io.Writer
	TaskQueue  TaskQueue
}

type EventQueue struct {
	Queue  chan struct{}
	mutex  sync.RWMutex
	closed bool
}

func (queue *EventQueue) Push() {
	queue.mutex.Lock()
	if !queue.closed {
		queue.Queue <- struct{}{}
	}
	queue.mutex.Unlock()
}

func (queue *EventQueue) Pop() {
	queue.mutex.Lock()
	<-queue.Queue
	queue.mutex.Unlock()
}

func (queue *EventQueue) Close() {
	queue.mutex.Lock()
	if !queue.closed {
		close(queue.Queue)
		queue.closed = true
	}
	queue.mutex.Unlock()
}

type TaskQueue struct {
	queue chan struct{}
}

func newTaskQueue(size int) TaskQueue {
	var queue chan struct{}
	if size > 0 {
		queue = make(chan struct{}, size)
	}
	return TaskQueue{
		queue: queue,
	}
}

func (queue *TaskQueue) push() {
	if queue.queue != nil {
		queue.queue <- struct{}{}
	}
}

func (queue *TaskQueue) pop() {
	if queue.queue != nil {
		<-queue.queue
	}
}

func (phase *Phase) Set(idx int, task Task) {
	phase.mutex.Lock()
	phase.Tasks[idx] = task
	phase.mutex.Unlock()
}

func (phase *Phase) GetAll() []Task {
	phase.mutex.RLock()
	a := phase.Tasks
	phase.mutex.RUnlock()
	return a
}

func (phase *Phase) Get(name string) []Task {
	arr := []Task{}
	phase.mutex.RLock()
	for _, task := range phase.Tasks {
		if task.Config.Name.Text == name {
			arr = append(arr, task)
		}
	}
	phase.mutex.RUnlock()
	return arr
}

func (phase ParamsPhase) outputResult(stderr io.Writer, name string) {
	fmt.Fprintln(stderr, "\n================")
	fmt.Fprintln(stderr, "= Phase Result: "+name+" =")
	fmt.Fprintln(stderr, "================")
	fmt.Fprintln(stderr, "status:", phase.Status)
	if phase.Error != nil {
		fmt.Fprintln(stderr, "error:", phase.Error)
	}
	utc := locale.UTC()
	runTasks := []Task{}
	for _, task := range phase.Tasks {
		if task.Result.Status == "skipped" {
			continue
		}
		runTasks = append(runTasks, task)
	}
	if len(runTasks) == 0 {
		fmt.Fprintln(stderr, "No task is run")
	}
	for _, task := range runTasks {
		fmt.Fprintln(stderr, "task:", task.Config.Name.Text)
		fmt.Fprintln(stderr, "status:", task.Result.Status)
		fmt.Fprintln(stderr, "exit code:", task.Result.Command.ExitCode)
		fmt.Fprintln(stderr, "start time:", task.Result.Time.Start.In(utc).Format(time.RFC3339))
		fmt.Fprintln(stderr, "end time:", task.Result.Time.End.In(utc).Format(time.RFC3339))
		fmt.Fprintln(stderr, "duration:", task.Result.Time.End.Sub(task.Result.Time.Start))
		fmt.Fprintln(stderr, task.Result.Command.CombinedOutput)
	}
}

func (phase *Phase) RunTask(ctx context.Context, idx int, task Task, params Params, wd string) error { //nolint:funlen,gocognit
	if task.Result.Status != constant.Queue {
		return nil
	}

	params.Item = task.Config.Item

	isFinished := true
	if task.Config.Dependency != nil {
		for _, dependOn := range task.Config.CompiledDependency.Names {
			dependencies := phase.Get(dependOn)
			if len(dependencies) == 0 {
				task.Result.Status = constant.Failed
				phase.Set(idx, task)
				return errors.New("invalid dependency. the task isn't found: " + dependOn)
			}
			for _, dependency := range dependencies {
				if !dependency.Result.IsFinished() {
					isFinished = false
				}
			}
		}
		b, err := task.Config.CompiledDependency.Program.Match(params.ToExpr())
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return fmt.Errorf("failed to evaluate the dependency: %w", err)
		}
		if !b {
			isFinished = false
		}
	}
	if !isFinished {
		return nil
	}

	params.Task = task

	f, err := task.Config.When.Match(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		phase.Set(idx, task)
		return fmt.Errorf(`failed to evaluate task's "when": %w`, err)
	}
	if !f {
		task.Result.Status = constant.Skipped
		phase.Set(idx, task)
		return nil
	}

	task.Result.Status = constant.Running
	phase.Set(idx, task)

	// evaluate input and add params
	input, err := task.Config.Input.Run(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		task.Result.Error = err
		logrus.WithFields(logrus.Fields{
			"phase_name": phase.Config.Name,
			"task_name":  task.Config.Name.Text,
			"task_index": idx,
		}).WithError(err).Error("failed to run an input")
		return fmt.Errorf(`failed to run an input: %w`, err)
	}
	task.Result.Input = input
	params.Task = task
	phase.Set(idx, task)

	switch task.Config.Type {
	case constant.Command:
		cmd, err := task.Config.Command.Command.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return fmt.Errorf(`failed to render a command: %w`, err)
		}
		task.Config.Command.Command = cmd

		m, err := renderEnvs(task.Config.Command.Env, params)
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return err
		}
		task.Config.Command.Env.Compiled = m

	case constant.ReadFile:
		p, err := task.Config.ReadFile.Path.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return fmt.Errorf(`failed to render read_file.path: %w`, err)
		}
		task.Config.ReadFile.Path = p
		if !filepath.IsAbs(p.Text) {
			task.Config.ReadFile.Path.Text = filepath.Join(wd, p.Text)
		}
	case constant.WriteFile:
		p, err := task.Config.WriteFile.Path.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return fmt.Errorf(`failed to render write_file.path: %w`, err)
		}
		task.Config.WriteFile.Path = p
		if !filepath.IsAbs(p.Text) {
			task.Config.WriteFile.Path.Text = filepath.Join(wd, p.Text)
		}
		tpl, err := task.Config.WriteFile.Template.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			phase.Set(idx, task)
			return fmt.Errorf(`failed to render write_file.template: %w`, err)
		}
		task.Config.WriteFile.Template = tpl
	}

	go func(idx int, task Task, params Params) {
		defer func() {
			phase.Set(idx, task)
			phase.EventQueue.Push()
		}()
		phase.TaskQueue.push()
		result, err := task.Run(ctx, wd)
		phase.TaskQueue.pop()
		task.Result = result
		if err != nil {
			task.Result.Status = constant.Failed
			task.Result.Error = err
			logrus.WithFields(logrus.Fields{
				"phase_name": phase.Config.Name,
				"task_name":  task.Config.Name.Text,
				"task_index": idx,
			}).WithError(err).Error("failed to run a task")
			return
		}
		task.Result.Status = constant.Succeeded
		params.Task = task
		phase.Set(idx, task)
		output, err := task.Config.Output.Run(params.ToExpr())
		if err != nil {
			task.Result.Status = constant.Failed
			task.Result.Error = err
			logrus.WithFields(logrus.Fields{
				"phase_name": phase.Config.Name,
				"task_name":  task.Config.Name.Text,
				"task_index": idx,
			}).WithError(err).Error("failed to run an output")
			return
		}
		task.Result.Output = output
	}(idx, task, params)
	return nil
}

func (phase *Phase) Run(ctx context.Context, params Params, wd string) error {
	for i, task := range phase.GetAll() {
		if err := phase.RunTask(ctx, i, task, params, wd); err != nil {
			logrus.WithFields(logrus.Fields{
				"task_name":  task.Config.Name.Text,
				"phase_name": phase.Config.Name,
			}).WithError(err).Error("failed to run a task")
		}
	}
	allFinished := true
	noRunning := true
	queuedTasks := []string{}
	for _, task := range phase.GetAll() {
		if !task.Result.IsFinished() {
			allFinished = false
			if task.Result.Status == constant.Running {
				noRunning = false
				break
			}
			if task.Result.Status == constant.Queue {
				queuedTasks = append(queuedTasks, task.Config.Name.Text)
			}
		}
	}
	if allFinished {
		phase.EventQueue.Close()
		return nil
	}
	if noRunning {
		return errors.New("the phase isn't finished but no task running. Plase check the task dependency is wrong. queued tasks: " + strings.Join(queuedTasks, ", "))
	}
	return nil
}
