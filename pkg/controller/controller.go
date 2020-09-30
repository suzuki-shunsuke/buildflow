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
	PR        interface{}
	Files     interface{}
	Util      map[string]interface{}
	Phases    map[string]ParamsPhase
	Tasks     []Task
	Task      Task
	PhaseName string
	Item      config.Item
}

type ParamsPhase struct {
	Tasks  []Task
	Status string
}

func (phase ParamsPhase) ToTemplate() map[string]interface{} {
	tasks := make([]map[string]interface{}, len(phase.Tasks))
	for i, task := range phase.Tasks {
		tasks[i] = task.ToTemplate()
	}
	return map[string]interface{}{
		"status": phase.Status,
		"tasks":  tasks,
	}
}

func (task Task) ToTemplate() map[string]interface{} {
	return map[string]interface{}{
		"name":            task.Config.Name.Text,
		"status":          task.Result.Status,
		"exit_code":       task.Result.Command.ExitCode,
		"stdout":          task.Result.Command.Stdout,
		"stderr":          task.Result.Command.Stderr,
		"combined_output": task.Result.Command.CombinedOutput,
		"file_text":       task.Result.File.Text,
	}
}

func (params Params) ToTemplate() interface{} {
	tasks := make([]map[string]interface{}, len(params.Tasks))
	for i, task := range params.Tasks {
		tasks[i] = task.ToTemplate()
	}
	phases := make(map[string]interface{}, len(params.Phases))
	for k, phase := range params.Phases {
		phases[k] = phase.ToTemplate()
	}
	return map[string]interface{}{
		"pr":    params.PR,
		"files": params.Files,
		"util":  params.Util,
		"task":  params.Task.ToTemplate(),
		// phases.<phase-name>.status
		// phases.<phase-name>.tasks[index].name
		// phases.<phase-name>.tasks[index].status
		"phases": phases,
		// phase.name
		"phase": map[string]interface{}{
			"name": params.PhaseName,
		},
		"tasks": tasks,
		"item": map[string]interface{}{
			"key":   params.Item.Key,
			"value": params.Item.Value,
		},
	}
}

func (params Params) ToExpr() interface{} {
	return params.ToTemplate()
}

func (ctrl Controller) newPhase(phaseCfg config.Phase) (Phase, error) { //nolint:unparam
	tasks := make([]Task, len(phaseCfg.Tasks))
	for i, taskCfg := range phaseCfg.Tasks {
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
	return Phase{
		Config: phaseCfg,
		Tasks:  tasks,
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

	for i, phaseCfg := range ctrl.Config.Phases {
		params.PhaseName = phaseCfg.Name
		tasksCfg := []config.Task{}
		for _, task := range phaseCfg.Tasks {
			tasks, err := Expand(task, params)
			if err != nil {
				return err
			}
			tasksCfg = append(tasksCfg, tasks...)
		}
		phaseCfg.Tasks = tasksCfg
		ctrl.Config.Phases[i] = phaseCfg

		if f, err := phaseCfg.Condition.Skip.Match(params.ToExpr()); err != nil {
			return err
		} else if f {
			// TODO update result
			continue
		}

		params.Phases = map[string]ParamsPhase{}

		if len(phaseCfg.Tasks) > 0 { //nolint:dupl
			phase, err := ctrl.newPhase(phaseCfg)
			if err != nil {
				return err
			}
			phase.EventQueue.Push()
			go func() {
				<-ctx.Done()
				phase.EventQueue.Close()
			}()
			params.Phases[phaseCfg.Name] = ParamsPhase{
				Tasks: phase.Tasks,
			}
			params.Tasks = phase.Tasks
			fmt.Fprintln(phase.Stderr, "\n==============")
			fmt.Fprintln(phase.Stderr, "= Phase: "+phaseCfg.Name+" =")
			fmt.Fprintln(phase.Stderr, "==============")
			for range phase.EventQueue.Queue {
				if err := phase.Run(ctx, params); err != nil {
					phase.EventQueue.Close()
					log.Println(err)
				}
				params.Phases[phaseCfg.Name] = ParamsPhase{
					Tasks: phase.Tasks,
				}
				params.Tasks = phase.Tasks
			}
			params.Phases[phaseCfg.Name] = ParamsPhase{
				Tasks: phase.Tasks,
			}
		}

		if f, err := phaseCfg.Condition.Exit.Match(params.ToExpr()); err != nil {
			return err
		} else if f {
			// TODO update result
			break
		}
	}
	return nil
}
