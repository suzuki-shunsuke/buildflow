package controller

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v32/github"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
	gh "github.com/suzuki-shunsuke/buildflow/pkg/github"
	"github.com/suzuki-shunsuke/go-dataeq/dataeq"
)

type Params struct {
	PR     interface{}
	Files  interface{}
	Util   map[string]interface{}
	Phases map[string][]Task
	Tasks  []Task
	Task   Task
	Item   config.Item
}

func (ctrl Controller) newTasks(taskCfgs []config.Task) (Tasks, error) { //nolint:unparam
	tasks := make([]Task, len(taskCfgs))
	for i, taskCfg := range taskCfgs {
		task := Task{
			Config: taskCfg,
			Result: domain.Result{
				Status: "queue",
			},
			Executor:   ctrl.Executor,
			Stdout:     execute.NewWriter(ctrl.Stdout, taskCfg.Name.Text),
			Stderr:     execute.NewWriter(ctrl.Stderr, taskCfg.Name.Text),
			Timer:      ctrl.Timer,
			FileReader: ctrl.FileReader,
		}
		tasks[i] = task
	}
	return Tasks{
		Tasks: tasks,
		EventQueue: EventQueue{
			Queue: make(chan struct{}, len(tasks)),
		},
		Stdout:    ctrl.Stdout,
		Stderr:    ctrl.Stderr,
		TaskQueue: newTaskQueue(ctrl.Config.Parallelism),
	}, nil
}

func (ctrl Controller) getPR(ctx context.Context) (*github.PullRequest, error) {
	if !ctrl.Config.PR {
		return nil, nil
	}
	if ctrl.Config.Env.PRNumber <= 0 {
		// get pull request from SHA
		prs, _, err := ctrl.GitHub.ListPRsWithCommit(ctx, gh.ParamsListPRsWithCommit{
			Owner: ctrl.Config.Owner,
			Repo:  ctrl.Config.Repo,
			SHA:   ctrl.Config.Env.SHA,
		})
		if err != nil {
			return nil, err
		}
		if len(prs) != 0 {
			return prs[0], nil
		}
		return nil, nil
	}
	pr, _, err := ctrl.GitHub.GetPR(ctx, gh.ParamsGetPR{
		Owner: ctrl.Config.Owner,
		Repo:  ctrl.Config.Repo,
		PRNum: ctrl.Config.Env.PRNumber,
	})
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (ctrl Controller) getTaskParams(ctx context.Context, pr *github.PullRequest) (Params, error) {
	if pr == nil {
		return Params{}, nil
	}
	prJSON, err := dataeq.JSON.Convert(pr)
	if err != nil {
		return Params{}, err
	}

	// get pull request files
	files, _, err := ctrl.GitHub.GetPRFiles(ctx, gh.ParamsGetPRFiles{
		Owner: ctrl.Config.Owner,
		Repo:  ctrl.Config.Repo,
		PRNum: *pr.Number,
	})
	if err != nil {
		return Params{}, err
	}
	filesJSON, err := dataeq.JSON.Convert(files)
	if err != nil {
		return Params{}, err
	}
	return Params{
		PR:    prJSON,
		Files: filesJSON,
	}, nil
}

func (ctrl Controller) Run(ctx context.Context) error { //nolint:funlen
	pr, err := ctrl.getPR(ctx)
	if err != nil {
		return err
	}
	params, err := ctrl.getTaskParams(ctx, pr)
	if err != nil {
		return err
	}
	params.Util = expr.GetUtil()

	for i, phase := range ctrl.Config.Phases {
		tasksCfg := []config.Task{}
		for _, task := range phase.Tasks {
			tasks, err := Expand(task, params)
			if err != nil {
				return err
			}
			tasksCfg = append(tasksCfg, tasks...)
		}
		phase.Tasks = tasksCfg
		ctrl.Config.Phases[i] = phase

		if f, err := phase.Condition.Skip.Match(params); err != nil {
			return err
		} else if f {
			// TODO update result
			continue
		}

		params.Phases = map[string][]Task{}

		if len(phase.Tasks) > 0 { //nolint:dupl
			tasks, err := ctrl.newTasks(phase.Tasks)
			if err != nil {
				return err
			}
			tasks.EventQueue.Push()
			go func() {
				<-ctx.Done()
				tasks.EventQueue.Close()
			}()
			params.Phases[phase.Name] = tasks.Tasks
			params.Tasks = tasks.Tasks
			fmt.Fprintln(tasks.Stderr, "\n==============")
			fmt.Fprintln(tasks.Stderr, "= Phase: "+phase.Name+" =")
			fmt.Fprintln(tasks.Stderr, "==============")
			for range tasks.EventQueue.Queue {
				if err := tasks.Run(ctx, params); err != nil {
					tasks.EventQueue.Close()
					log.Println(err)
				}
				params.Phases[phase.Name] = tasks.Tasks
				params.Tasks = tasks.Tasks
			}
			params.Phases[phase.Name] = tasks.Tasks
		}

		if f, err := phase.Condition.Exit.Match(params); err != nil {
			return err
		} else if f {
			// TODO update result
			break
		}
	}
	return nil
}
