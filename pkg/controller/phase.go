package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
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

func (phase *Phase) RunTask(ctx context.Context, idx int, task Task, params Params) error { //nolint:funlen,gocognit
	if task.Result.Status != domain.TaskResultQueue {
		return nil
	}

	params.Item = task.Config.Item

	isFinished := true
	if task.Config.Dependency != nil {
		for _, dependOn := range task.Config.CompiledDependency.Names {
			dependencies := phase.Get(dependOn)
			if len(dependencies) == 0 {
				task.Result.Status = domain.TaskResultFailed
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
			task.Result.Status = domain.TaskResultFailed
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
		task.Result.Status = domain.TaskResultFailed
		phase.Set(idx, task)
		return err
	}
	if !f {
		task.Result.Status = domain.TaskResultSkipped
		phase.Set(idx, task)
		return nil
	}

	task.Result.Status = domain.TaskResultRunning
	phase.Set(idx, task)

	switch task.Config.Type {
	case domain.TaskTypeCommand:
		cmd, err := task.Config.Command.Command.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			phase.Set(idx, task)
			return err
		}
		task.Config.Command.Command = cmd

		m, err := renderEnvs(task.Config.Command.Env, params)
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			phase.Set(idx, task)
			return err
		}
		task.Config.Command.Env.Compiled = m

	case domain.TaskTypeFile:
		p, err := task.Config.ReadFile.Path.New(params.ToTemplate())
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			phase.Set(idx, task)
			return err
		}
		task.Config.ReadFile.Path = p
	}

	go func(idx int, task Task, params Params) {
		defer func() {
			phase.Set(idx, task)
			phase.EventQueue.Push()
		}()
		phase.TaskQueue.push()
		result, err := task.Run(ctx)
		phase.TaskQueue.pop()
		task.Result = result
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			task.Result.Error = err
			log.Println(err)
			return
		}
		task.Result.Status = domain.TaskResultSucceeded
		params.Task = task
		phase.Set(idx, task)
		output, err := task.Config.Output.Run(params.ToTemplate())
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
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

func (phase *Phase) Run(ctx context.Context, params Params) error {
	for i, task := range phase.GetAll() {
		if err := phase.RunTask(ctx, i, task, params); err != nil {
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
			if task.Result.Status == domain.TaskResultRunning {
				noRunning = false
				break
			}
			if task.Result.Status == domain.TaskResultQueue {
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
