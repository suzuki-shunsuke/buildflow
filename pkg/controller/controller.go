package controller

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	gh "github.com/suzuki-shunsuke/buildflow/pkg/github"
	"github.com/suzuki-shunsuke/buildflow/pkg/util"
	"github.com/suzuki-shunsuke/go-dataeq/dataeq"
)

type Params struct {
	PR        interface{}
	Files     interface{}
	Util      map[string]interface{}
	Phases    map[string]ParamsPhase
	Task      Task
	PhaseName string
	Item      config.Item
	Meta      map[string]interface{}
}

type ParamsPhase struct {
	Tasks  []Task
	Status string
	Error  error
	Exit   bool
	Meta   map[string]interface{}
	Name   string
}

func (phase ParamsPhase) ToTemplate() map[string]interface{} {
	tasks := make([]map[string]interface{}, len(phase.Tasks))
	for i, task := range phase.Tasks {
		tasks[i] = task.ToTemplate()
	}
	return map[string]interface{}{
		"Status": phase.Status,
		"Tasks":  tasks,
		"Meta":   phase.Meta,
		"Name":   phase.Name,
	}
}

func (task Task) ToTemplate() map[string]interface{} {
	return map[string]interface{}{
		"Name":           task.Config.Name.Text,
		"Status":         task.Result.Status,
		"ExitCode":       task.Result.Command.ExitCode,
		"Stdout":         task.Result.Command.Stdout,
		"Stderr":         task.Result.Command.Stderr,
		"CombinedOutput": task.Result.Command.CombinedOutput,
		"FileText":       task.Result.File.Text,
		"Meta":           task.Config.Meta,
		"Outputs":        task.Result.Output,
	}
}

func (params Params) ToTemplate() map[string]interface{} {
	phases := make(map[string]interface{}, len(params.Phases))
	for k, phase := range params.Phases {
		phases[k] = phase.ToTemplate()
	}
	m := map[string]interface{}{
		"PR":    params.PR,
		"Files": params.Files,
		"Task":  params.Task.ToTemplate(),
		// phases.<phase-name>.status
		// phases.<phase-name>.tasks[index].name
		// phases.<phase-name>.tasks[index].status
		"Phases": phases,
		"Item": map[string]interface{}{
			"Key":   params.Item.Key,
			"Value": params.Item.Value,
		},
		"Meta": params.Meta,
	}

	var tasks []map[string]interface{}
	if params.PhaseName != "" {
		pTasks := params.Phases[params.PhaseName].Tasks
		tasks = make([]map[string]interface{}, len(pTasks))
		for i, task := range pTasks {
			tasks[i] = task.ToTemplate()
		}

		m["Phase"] = params.Phases[params.PhaseName].ToTemplate()
	}
	m["Tasks"] = tasks
	return m
}

func (params Params) ToExpr() map[string]interface{} {
	a := params.ToTemplate()
	a["Util"] = params.Util
	return a
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
		logrus.Debug("pr is disabled")
		return nil, nil
	}
	prNum := ctrl.Config.Env.PRNumber
	if prNum <= 0 {
		logrus.WithFields(logrus.Fields{
			"owner": ctrl.Config.Owner,
			"repo":  ctrl.Config.Repo,
			"sha":   ctrl.Config.Env.SHA,
		}).Debug("get pull request from SHA")
		prs, _, err := ctrl.GitHub.ListPRsWithCommit(ctx, gh.ParamsListPRsWithCommit{
			Owner: ctrl.Config.Owner,
			Repo:  ctrl.Config.Repo,
			SHA:   ctrl.Config.Env.SHA,
		})
		if err != nil {
			return nil, err
		}
		logrus.WithFields(logrus.Fields{
			"size": len(prs),
		}).Debug("the number of pull requests assosicated with the commit")
		if len(prs) == 0 {
			return nil, nil
		}
		prNum = prs[0].GetNumber()
	}
	pr, _, err := ctrl.GitHub.GetPR(ctx, gh.ParamsGetPR{
		Owner: ctrl.Config.Owner,
		Repo:  ctrl.Config.Repo,
		PRNum: prNum,
	})
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (ctrl Controller) getTaskParams(ctx context.Context, pr *github.PullRequest) (Params, error) {
	params := Params{
		Util:   util.GetUtil(),
		Meta:   ctrl.Config.Meta,
		Phases: make(map[string]ParamsPhase, len(ctrl.Config.Phases)),
	}
	for _, phase := range ctrl.Config.Phases {
		params.Phases[phase.Name] = ParamsPhase{
			Meta: phase.Meta,
			Name: phase.Name,
		}
	}

	if pr == nil {
		logrus.Debug("pr is nil")
		return params, nil
	}
	prJSON, err := dataeq.JSON.Convert(pr)
	if err != nil {
		return params, err
	}
	params.PR = prJSON

	// get pull request files
	files, _, err := ctrl.GitHub.GetPRFiles(ctx, gh.ParamsGetPRFiles{
		Owner:    ctrl.Config.Owner,
		Repo:     ctrl.Config.Repo,
		PRNum:    pr.GetNumber(),
		FileSize: pr.GetChangedFiles(),
	})
	logrus.WithFields(logrus.Fields{
		"files_gotten_by_api": len(files),
		"changed_files":       pr.GetChangedFiles(),
	}).Debug("the number of pull request files")
	if err != nil {
		return params, err
	}
	filesJSON, err := dataeq.JSON.Convert(files)
	if err != nil {
		return params, err
	}
	params.Files = filesJSON

	return params, nil
}

func (ctrl Controller) runPhase(ctx context.Context, params Params, idx int) (ParamsPhase, error) { //nolint:funlen
	phaseCfg := ctrl.Config.Phases[idx]
	params.PhaseName = phaseCfg.Name
	tasksCfg := []config.Task{}
	phaseParams := ParamsPhase{
		Meta: phaseCfg.Meta,
		Name: phaseCfg.Name,
	}
	for _, task := range phaseCfg.Tasks {
		tasks, err := Expand(task, params)
		if err != nil {
			phaseParams.Error = err
			break
		}
		tasksCfg = append(tasksCfg, tasks...)
	}
	if phaseParams.Error != nil {
		return phaseParams, nil
	}
	phaseCfg.Tasks = tasksCfg
	ctrl.Config.Phases[idx] = phaseCfg

	if f, err := phaseCfg.Condition.Skip.Match(params.ToExpr()); err != nil {
		phaseParams.Error = err
		return phaseParams, nil
	} else if f {
		phaseParams.Status = domain.ResultSkipped
		return phaseParams, nil
	}

	if len(phaseCfg.Tasks) > 0 { //nolint:dupl
		phase, err := ctrl.newPhase(phaseCfg)
		if err != nil {
			phaseParams.Error = err
			return phaseParams, nil
		}
		phaseParams.Tasks = phase.Tasks
		phase.EventQueue.Push()
		go func() {
			<-ctx.Done()
			phase.EventQueue.Close()
		}()
		params.Phases[phaseCfg.Name] = phaseParams
		fmt.Fprintln(phase.Stderr, "\n==============")
		fmt.Fprintln(phase.Stderr, "= Phase: "+phaseCfg.Name+" =")
		fmt.Fprintln(phase.Stderr, "==============")
		for range phase.EventQueue.Queue {
			if err := phase.Run(ctx, params); err != nil {
				phase.EventQueue.Close()
				log.Println(err)
			}
			phaseParams.Tasks = phase.Tasks
			params.Phases[phaseCfg.Name] = phaseParams
		}
		phaseParams.Tasks = phase.Tasks
		params.Phases[phaseCfg.Name] = phaseParams
		phaseParams.Tasks = phase.Tasks
	}

	if f, err := phaseCfg.Condition.Fail.Match(params.ToExpr()); err != nil { //nolint:gocritic
		phaseParams.Error = err
		return phaseParams, nil
	} else if f {
		phaseParams.Status = domain.ResultFailed
	} else {
		phaseParams.Status = domain.ResultSucceeded
	}

	if f, err := phaseCfg.Condition.Exit.Match(params.ToExpr()); err != nil {
		return phaseParams, err
	} else if f {
		// TODO update result
		phaseParams.Exit = true
		return phaseParams, nil
	}
	return phaseParams, nil
}

var ErrBuildFail = errors.New("build failed")

func (ctrl Controller) Run(ctx context.Context) error { //nolint:funlen,gocognit
	pr, err := ctrl.getPR(ctx)
	if err != nil {
		return err
	}

	if pr != nil {
		logrus.WithFields(logrus.Fields{
			"pr_number":     pr.GetNumber(),
			"changed_files": pr.GetChangedFiles(),
		}).Debug("pull request")
	}

	params, err := ctrl.getTaskParams(ctx, pr)
	if err != nil {
		return err
	}

	if f, err := ctrl.Config.Condition.Skip.Match(params.ToExpr()); err != nil {
		return err
	} else if f {
		fmt.Fprintln(ctrl.Stderr, "the build is skipped")
		return nil
	}

	for i, phaseCfg := range ctrl.Config.Phases {
		phaseParams, err := ctrl.runPhase(ctx, params, i)
		if phaseParams.Error != nil {
			phaseParams.Status = domain.ResultFailed
		}
		phaseParams.outputResult(ctrl.Stderr, phaseCfg.Name)
		params.Phases[phaseCfg.Name] = phaseParams
		if err != nil {
			return err
		}
		if phaseParams.Exit {
			break
		}
	}

	if f, err := ctrl.Config.Condition.Fail.Match(params.ToExpr()); err != nil {
		return err
	} else if f {
		return ErrBuildFail
	}

	return nil
}
