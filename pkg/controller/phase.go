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
	Queue chan struct{}
	once  sync.Once
}

func (queue *EventQueue) Push() {
	queue.Queue <- struct{}{}
}

func (queue *EventQueue) Pop() {
	<-queue.Queue
}

func (queue *EventQueue) Close() {
	queue.once.Do(func() {
		close(queue.Queue)
	})
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

func (phase *Phase) outputResult() {
	fmt.Fprintln(phase.Stderr, "\n================")
	fmt.Fprintln(phase.Stderr, "= Phase Result =")
	fmt.Fprintln(phase.Stderr, "================")
	utc := locale.UTC()
	runTasks := []Task{}
	for _, task := range phase.GetAll() {
		if task.Result.Status == "skipped" {
			continue
		}
		runTasks = append(runTasks, task)
	}
	if len(runTasks) == 0 {
		fmt.Fprintln(phase.Stderr, "No task is run")
	}
	for _, task := range runTasks {
		fmt.Fprintln(phase.Stderr, "task:", task.Config.Name.Text)
		fmt.Fprintln(phase.Stderr, "status:", task.Result.Status)
		fmt.Fprintln(phase.Stderr, "exit code:", task.Result.Command.ExitCode)
		fmt.Fprintln(phase.Stderr, "start time:", task.Result.Time.Start.In(utc).Format(time.RFC3339))
		fmt.Fprintln(phase.Stderr, "end time:", task.Result.Time.End.In(utc).Format(time.RFC3339))
		fmt.Fprintln(phase.Stderr, "duration:", task.Result.Time.End.Sub(task.Result.Time.Start))
		fmt.Fprintln(phase.Stderr, task.Result.Command.CombinedOutput)
	}
}

func (phase *Phase) RunTask(ctx context.Context, idx int, task Task, params Params) error { //nolint:funlen
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
			return err
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

	go func(idx int, task Task) {
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
			log.Println(err)
			return
		}
		task.Result.Status = domain.TaskResultSucceeded
	}(idx, task)
	return nil
}

func (phase *Phase) Run(ctx context.Context, params Params) error {
	for i, task := range phase.GetAll() {
		if err := phase.RunTask(ctx, i, task, params); err != nil {
			fmt.Fprintln(phase.Stderr, "task: "+task.Config.Name.Text, err)
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
		phase.outputResult()
		phase.EventQueue.Close()
		return nil
	}
	if noRunning {
		return errors.New("the phase isn't finished but no task running. Plase check the task dependency is wrong. queued tasks: " + strings.Join(queuedTasks, ", "))
	}
	return nil
}
