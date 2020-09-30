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

	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/locale"
)

type Tasks struct {
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

func (tasks *Tasks) Set(idx int, task Task) {
	tasks.mutex.Lock()
	tasks.Tasks[idx] = task
	tasks.mutex.Unlock()
}

func (tasks *Tasks) GetAll() []Task {
	tasks.mutex.RLock()
	a := tasks.Tasks
	tasks.mutex.RUnlock()
	return a
}

func (tasks *Tasks) Get(name string) []Task {
	arr := []Task{}
	tasks.mutex.RLock()
	for _, task := range tasks.Tasks {
		if task.Config.Name.Text == name {
			arr = append(arr, task)
		}
	}
	tasks.mutex.RUnlock()
	return arr
}

func (tasks *Tasks) outputResult() {
	fmt.Fprintln(tasks.Stderr, "\n================")
	fmt.Fprintln(tasks.Stderr, "= Phase Result =")
	fmt.Fprintln(tasks.Stderr, "================")
	utc := locale.UTC()
	runTasks := []Task{}
	for _, task := range tasks.GetAll() {
		if task.Result.Status == "skipped" {
			continue
		}
		runTasks = append(runTasks, task)
	}
	if len(runTasks) == 0 {
		fmt.Fprintln(tasks.Stderr, "No task is run")
	}
	for _, task := range runTasks {
		fmt.Fprintln(tasks.Stderr, "task:", task.Config.Name.Text)
		fmt.Fprintln(tasks.Stderr, "status:", task.Result.Status)
		fmt.Fprintln(tasks.Stderr, "exit code:", task.Result.Command.ExitCode)
		fmt.Fprintln(tasks.Stderr, "start time:", task.Result.Time.Start.In(utc).Format(time.RFC3339))
		fmt.Fprintln(tasks.Stderr, "end time:", task.Result.Time.End.In(utc).Format(time.RFC3339))
		fmt.Fprintln(tasks.Stderr, "duration:", task.Result.Time.End.Sub(task.Result.Time.Start))
		fmt.Fprintln(tasks.Stderr, task.Result.Command.CombinedOutput)
	}
}

func (tasks *Tasks) RunTask(ctx context.Context, idx int, task Task, params Params) error { //nolint:funlen
	if task.Result.Status != domain.TaskResultQueue {
		return nil
	}

	params.Item = task.Config.Item

	isFinished := true
	if task.Config.Dependency != nil {
		for _, dependOn := range task.Config.CompiledDependency.Names {
			dependencies := tasks.Get(dependOn)
			if len(dependencies) == 0 {
				task.Result.Status = domain.TaskResultFailed
				tasks.Set(idx, task)
				return errors.New("invalid dependency. the task isn't found: " + dependOn)
			}
			for _, dependency := range dependencies {
				if !dependency.Result.IsFinished() {
					isFinished = false
				}
			}
		}
		b, err := task.Config.CompiledDependency.Program.Match(params)
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			tasks.Set(idx, task)
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

	f, err := task.Config.When.Match(params)
	if err != nil {
		task.Result.Status = domain.TaskResultFailed
		tasks.Set(idx, task)
		return err
	}
	if !f {
		task.Result.Status = domain.TaskResultSkipped
		tasks.Set(idx, task)
		return nil
	}

	task.Result.Status = domain.TaskResultRunning
	tasks.Set(idx, task)

	switch task.Config.Type {
	case domain.TaskTypeCommand:
		cmd, err := task.Config.Command.Command.New(params)
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			tasks.Set(idx, task)
			return err
		}
		task.Config.Command.Command = cmd
	case domain.TaskTypeFile:
		p, err := task.Config.ReadFile.Path.New(params)
		if err != nil {
			task.Result.Status = domain.TaskResultFailed
			tasks.Set(idx, task)
			return err
		}
		task.Config.ReadFile.Path = p
	}

	go func(idx int, task Task) {
		defer func() {
			tasks.Set(idx, task)
			tasks.EventQueue.Push()
		}()
		tasks.TaskQueue.push()
		result, err := task.Run(ctx)
		tasks.TaskQueue.pop()
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

func (tasks *Tasks) Run(ctx context.Context, params Params) error {
	for i, task := range tasks.GetAll() {
		if err := tasks.RunTask(ctx, i, task, params); err != nil {
			fmt.Fprintln(tasks.Stderr, "task: "+task.Config.Name.Text, err)
		}
	}
	allFinished := true
	noRunning := true
	queuedTasks := []string{}
	for _, task := range tasks.GetAll() {
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
		tasks.outputResult()
		tasks.EventQueue.Close()
		return nil
	}
	if noRunning {
		return errors.New("the phase isn't finished but no task running. Plase check the task dependency is wrong. queued tasks: " + strings.Join(queuedTasks, ", "))
	}
	return nil
}
