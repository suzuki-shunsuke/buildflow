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
	EventQueue *EventQueue
	Stdout     io.Writer
	Stderr     io.Writer
	TaskQueue  TaskQueue
	Status     string
	Error      error
	Exit       bool
	Tasks      *TaskList
}

func (phase Phase) Name() string {
	return phase.Config.Name
}

func (phase Phase) Meta() map[string]interface{} {
	return phase.Config.Meta
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

func (phase *Phase) Get(name string) []Task {
	arr := []Task{}
	for _, task := range phase.Tasks.GetAll() {
		if task.Name() == name {
			arr = append(arr, task)
		}
	}
	return arr
}

func (phase Phase) outputResult(stderr io.Writer, name string) {
	fmt.Fprintln(stderr, "\n================")
	fmt.Fprintln(stderr, "= Phase Result: "+name+" =")
	fmt.Fprintln(stderr, "================")
	fmt.Fprintln(stderr, "status:", phase.Status)
	if phase.Error != nil {
		fmt.Fprintln(stderr, "error:", phase.Error)
	}
	utc := locale.UTC()
	runTasks := []Task{}
	for _, task := range phase.Tasks.GetAll() {
		if task.Result.Status == "skipped" {
			continue
		}
		runTasks = append(runTasks, task)
	}
	if len(runTasks) == 0 {
		fmt.Fprintln(stderr, "No task is run")
	}
	for _, task := range runTasks {
		fmt.Fprintln(stderr, "task:", task.Name())
		fmt.Fprintln(stderr, "status:", task.Result.Status)
		fmt.Fprintln(stderr, "exit code:", task.Result.Command.ExitCode)
		fmt.Fprintln(stderr, "start time:", task.Result.Time.Start.In(utc).Format(time.RFC3339))
		fmt.Fprintln(stderr, "end time:", task.Result.Time.End.In(utc).Format(time.RFC3339))
		fmt.Fprintln(stderr, "duration:", task.Result.Time.End.Sub(task.Result.Time.Start))
		fmt.Fprintln(stderr, task.Result.Command.CombinedOutput)
	}
}

func (phase *Phase) IsReady(task Task, params Params) (bool, error) {
	for _, dependOn := range task.Config.Dependency.Names {
		dependencies := phase.Get(dependOn)
		if len(dependencies) == 0 {
			task.Result.Status = constant.Failed
			return false, errors.New("invalid dependency. the task isn't found: " + dependOn)
		}
		for _, dependency := range dependencies {
			if !dependency.Result.IsFinished() {
				return false, nil
			}
		}
	}
	b, err := task.Config.Dependency.Program.Match(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		return false, fmt.Errorf("failed to evaluate the dependency: %w", err)
	}
	if !b {
		return false, nil
	}
	return true, nil
}

func (phase *Phase) PrepareCommandTask(task Task, params Params, wd string) (Task, error) {
	cmd, err := task.Config.Command.Command.New(params.ToTemplate())
	if err != nil {
		task.Result.Status = constant.Failed
		return task, fmt.Errorf(`failed to render a command: %w`, err)
	}
	task.Config.Command.Command = cmd

	stdin, err := task.Config.Command.Stdin.New(params.ToTemplate())
	if err != nil {
		task.Result.Status = constant.Failed
		return task, fmt.Errorf(`failed to render a command.stdin: %w`, err)
	}
	task.Config.Command.Stdin = stdin

	m, err := renderEnvs(task.Config.Command.Env, params)
	if err != nil {
		task.Result.Status = constant.Failed
		return task, err
	}
	task.Config.Command.Env.Compiled = m
	return task, nil
}

func (phase *Phase) PrepareTask(task Task, params Params, wd string) (Task, error) {
	switch task.Config.Type {
	case constant.Command:
		return phase.PrepareCommandTask(task, params, wd)
	case constant.ReadFile:
		p, err := task.Config.ReadFile.Path.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			return task, fmt.Errorf(`failed to render read_file.path: %w`, err)
		}
		task.Config.ReadFile.Path = p
		if !filepath.IsAbs(p.Text) {
			task.Config.ReadFile.Path.Text = filepath.Join(wd, p.Text)
		}
		return task, nil
	case constant.WriteFile:
		p, err := task.Config.WriteFile.Path.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			return task, fmt.Errorf(`failed to render write_file.path: %w`, err)
		}
		task.Config.WriteFile.Path = p
		if !filepath.IsAbs(p.Text) {
			task.Config.WriteFile.Path.Text = filepath.Join(wd, p.Text)
		}
		tpl, err := task.Config.WriteFile.Template.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = constant.Failed
			return task, fmt.Errorf(`failed to render write_file.template: %w`, err)
		}
		task.Config.WriteFile.Template = tpl
		return task, nil
	default:
		return task, errors.New("invalid task type")
	}
}

func (phase *Phase) runTask(ctx context.Context, idx int, task Task, params Params, paramsPhase Phase, wd string) {
	defer func() {
		paramsPhase.Tasks.Set(idx, task)
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
			"task_name":  task.Name(),
			"task_index": idx,
		}).WithError(err).Error("failed to run a task")
		return
	}
	task.Result.Status = constant.Succeeded
	output, err := task.Config.Output.Run(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		task.Result.Error = err
		logrus.WithFields(logrus.Fields{
			"phase_name": phase.Config.Name,
			"task_name":  task.Name(),
			"task_index": idx,
		}).WithError(err).Error("failed to run an output")
		return
	}
	task.Result.Output = output
}

func (phase *Phase) RunTask(ctx context.Context, idx int, task Task, params Params, wd string) error {
	if task.Result.Status != constant.Queue {
		return nil
	}
	params.TaskIdx = idx

	params.Item = task.Config.Item
	paramsPhase := params.Phases[params.PhaseName]
	defer func() {
		paramsPhase.Tasks.Set(idx, task)
	}()

	if isReady, err := phase.IsReady(task, params); err != nil || !isReady {
		return err
	}

	f, err := task.Config.When.Match(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		return fmt.Errorf(`failed to evaluate task's "when": %w`, err)
	}
	if !f {
		task.Result.Status = constant.Skipped
		return nil
	}

	task.Result.Status = constant.Running
	paramsPhase.Tasks.Set(idx, task)

	// evaluate input and add params
	input, err := task.Config.Input.Run(params.ToExpr())
	if err != nil {
		task.Result.Status = constant.Failed
		task.Result.Error = err
		logrus.WithFields(logrus.Fields{
			"phase_name": phase.Config.Name,
			"task_name":  task.Name(),
			"task_index": idx,
		}).WithError(err).Error("failed to run an input")
		return fmt.Errorf(`failed to run an input: %w`, err)
	}
	task.Result.Input = input
	paramsPhase.Tasks.Set(idx, task)

	task, err = phase.PrepareTask(task, params, wd)
	if err != nil {
		return err
	}

	go phase.runTask(ctx, idx, task, params, paramsPhase, wd)
	return nil
}

func (phase *Phase) Run(ctx context.Context, params Params, wd string) error {
	p := params.Phases[params.PhaseName]
	for i, task := range p.Tasks.GetAll() {
		if err := phase.RunTask(ctx, i, task, params, wd); err != nil {
			logrus.WithFields(logrus.Fields{
				"task_name":  task.Name(),
				"phase_name": phase.Config.Name,
			}).WithError(err).Error("failed to run a task")
		}
	}
	allFinished := true
	noRunning := true
	queuedTasks := []string{}
	for _, task := range phase.Tasks.GetAll() {
		if !task.Result.IsFinished() {
			allFinished = false
			if task.Result.Status == constant.Running {
				noRunning = false
				break
			}
			if task.Result.Status == constant.Queue {
				queuedTasks = append(queuedTasks, task.Name())
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
