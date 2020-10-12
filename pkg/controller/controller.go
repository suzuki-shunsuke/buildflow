package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"github.com/suzuki-shunsuke/buildflow/pkg/config"
	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
	"github.com/suzuki-shunsuke/buildflow/pkg/expr"
	gh "github.com/suzuki-shunsuke/buildflow/pkg/github"
	"github.com/suzuki-shunsuke/buildflow/pkg/template"
	"github.com/suzuki-shunsuke/go-dataeq/dataeq"
)

type Params struct {
	PR        interface{}
	Files     interface{}
	Phases    map[string]Phase
	TaskIdx   int
	PhaseName string
	Item      config.Item
	Meta      map[string]interface{}
}

type TaskList struct {
	tasks []Task
	mutex sync.RWMutex
}

func (list *TaskList) Set(idx int, task Task) {
	list.mutex.Lock()
	list.tasks[idx] = task
	list.mutex.Unlock()
}

func (list *TaskList) GetAll() []Task {
	list.mutex.RLock()
	arr := make([]Task, len(list.tasks))
	for i, task := range list.tasks {
		arr[i] = task
	}
	list.mutex.RUnlock()
	return arr
}

func (list *TaskList) Size() int {
	list.mutex.RLock()
	n := len(list.tasks)
	list.mutex.RUnlock()
	return n
}

func (list *TaskList) Get(idx int) Task {
	if idx < 0 {
		return Task{}
	}
	list.mutex.RLock()
	task := list.tasks[idx]
	list.mutex.RUnlock()
	return task
}

func (phase Phase) ToTemplate() map[string]interface{} {
	tasks := make([]interface{}, phase.Tasks.Size())
	for i, task := range phase.Tasks.GetAll() {
		tasks[i] = task.ToTemplate()
	}
	return map[string]interface{}{
		"Status": phase.Status,
		"Tasks":  tasks,
		"Meta":   phase.Meta(),
		"Name":   phase.Name(),
	}
}

func (task Task) Name() string {
	return task.Config.Name.Text
}

func (task Task) ToTemplate() map[string]interface{} {
	m := map[string]interface{}{
		"Name":   task.Name(),
		"Type":   task.Config.Type,
		"Status": task.Result.Status,
		"Meta":   task.Config.Meta,
		"Output": task.Result.Output,
		"Input":  task.Result.Input,
	}
	switch task.Config.Type {
	case constant.Command:
		m["ExitCode"] = task.Result.Command.ExitCode
		m["Stdout"] = task.Result.Command.Stdout
		m["Stderr"] = task.Result.Command.Stderr
		m["CombinedOutput"] = task.Result.Command.CombinedOutput
	case constant.HTTP:
	case constant.ReadFile, constant.WriteFile:
		m["File"] = task.Result.File.ToTemplate()
	}
	return m
}

func (params Params) ToTemplate() map[string]interface{} {
	phases := make(map[string]interface{}, len(params.Phases))
	for k, phase := range params.Phases {
		phases[k] = phase.ToTemplate()
	}
	task := Task{}
	if params.PhaseName != "" {
		phase, ok := params.Phases[params.PhaseName]
		if !ok {
			panic("phase not found: " + params.PhaseName)
		}
		task = phase.Tasks.Get(params.TaskIdx)
	}
	m := map[string]interface{}{
		"PR":    params.PR,
		"Files": params.Files,
		"Task":  task.ToTemplate(),
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

	var tasks []interface{}
	if params.PhaseName != "" {
		t := params.Phases[params.PhaseName].Tasks
		pTasks := t.GetAll()
		tasks = make([]interface{}, len(pTasks))
		for i, task := range pTasks {
			tasks[i] = task.ToTemplate()
		}

		m["Phase"] = params.Phases[params.PhaseName].ToTemplate()
	}
	m["Tasks"] = tasks
	return m
}

func (params Params) ToExpr() map[string]interface{} {
	return params.ToTemplate()
}

func (ctrl Controller) newPhase(phaseCfg config.Phase) Phase {
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
			FileWriter: ctrl.FileWriter,
		}
		tasks[i] = task
	}
	return Phase{
		Config: phaseCfg,
		Tasks: &TaskList{
			tasks: tasks,
		},
		EventQueue: &EventQueue{
			Queue: make(chan struct{}, len(tasks)),
		},
		Stdout:    ctrl.Stdout,
		Stderr:    ctrl.Stderr,
		TaskQueue: newTaskQueue(ctrl.Config.Parallelism),
	}
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

func (ctrl Controller) getParams(ctx context.Context, pr *github.PullRequest) (Params, error) {
	params := Params{
		Meta:    ctrl.Config.Meta,
		Phases:  make(map[string]Phase, len(ctrl.Config.Phases)),
		TaskIdx: -1,
	}
	for _, phase := range ctrl.Config.Phases {
		params.Phases[phase.Name] = Phase{
			Tasks: &TaskList{
				tasks: []Task{},
			},
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

func (ctrl Controller) getPhase(params Params, phaseCfg config.Phase) Phase {
	params.PhaseName = phaseCfg.Name
	phase := Phase{
		Tasks: &TaskList{
			tasks: []Task{},
		},
		Config: phaseCfg,
	}

	tasksCfg := []config.Task{}
	for _, task := range phaseCfg.Tasks {
		tasks, err := Expand(task, params)
		if err != nil {
			phase.Error = err
			break
		}
		tasksCfg = append(tasksCfg, tasks...)
	}
	if phase.Error != nil {
		return phase
	}
	phaseCfg.Tasks = tasksCfg
	return ctrl.newPhase(phaseCfg)
}

func (ctrl Controller) checkSkipPhase(params Params, phase Phase, phaseCfg config.Phase) (Phase, bool) {
	if f, err := phaseCfg.Condition.Skip.Match(params.ToExpr()); err != nil {
		logrus.WithFields(logrus.Fields{
			"phase_name": phaseCfg.Name,
		}).WithError(err).Error(`failed to evaluate the phase's skip condition`)
		phase.Error = err
		phase.Status = constant.Failed
		return phase, true
	} else if f {
		phase.Status = constant.Skipped
		return phase, true
	}
	return phase, false
}

func (ctrl Controller) printPhaseHeader(phase Phase) {
	fmt.Fprintln(phase.Stderr, "\n==============")
	fmt.Fprintln(phase.Stderr, "= Phase: "+phase.Config.Name+" =")
	fmt.Fprintln(phase.Stderr, "==============")
}

func (ctrl Controller) runPhase(ctx context.Context, params Params, idx int, wd string) (Phase, error) {
	phaseCfg := ctrl.Config.Phases[idx]
	if len(phaseCfg.Tasks) == 0 {
		return Phase{}, nil
	}
	phase := ctrl.getPhase(params, phaseCfg)
	if phase.Error != nil {
		return phase, nil
	}
	params.PhaseName = phase.Name()
	params.Phases[params.PhaseName] = phase

	if p, f := ctrl.checkSkipPhase(params, phase, phaseCfg); f {
		return phase, nil
	} else {
		phase = p
	}

	phase.EventQueue.Push()
	go func() {
		<-ctx.Done()
		phase.EventQueue.Close()
	}()
	params.Phases[phaseCfg.Name] = phase
	ctrl.printPhaseHeader(phase)
	for range phase.EventQueue.Queue {
		if err := phase.Run(ctx, params, wd); err != nil {
			phase.EventQueue.Close()
			log.Println(err)
		}
		params.Phases[phaseCfg.Name] = phase
	}
	params.Phases[phaseCfg.Name] = phase

	if f, err := phaseCfg.Condition.Fail.Match(params.ToExpr()); err != nil { //nolint:gocritic
		phase.Error = err
		return phase, nil
	} else if f {
		phase.Status = constant.Failed
	} else {
		phase.Status = constant.Succeeded
	}

	if f, err := phaseCfg.Condition.Exit.Match(params.ToExpr()); err != nil {
		return phase, err
	} else if f {
		// TODO update result
		phase.Exit = true
		return phase, nil
	}
	return phase, nil
}

var ErrBuildFail = errors.New("build failed")

func (ctrl Controller) readTemplateFile(p, wd string, tpl *config.Template) error {
	if p == "" {
		return nil
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(wd, p)
	}
	result, err := ctrl.FileReader.Read(p)
	if err != nil {
		return err
	}
	if err := tpl.SetText(result.Text); err != nil {
		return err
	}
	return nil
}

func (ctrl Controller) readScript(p, wd string, scr *config.Script) error {
	if p == "" {
		return nil
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(wd, p)
	}
	result, err := ctrl.FileReader.Read(p)
	if err != nil {
		return err
	}
	if prog, err := expr.New(result.Text); err != nil {
		return err
	} else {
		scr.Prog = prog
	}
	return nil
}

func (ctrl Controller) readBoolScript(p, wd string, scr *config.Bool) error {
	if p == "" {
		return nil
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(wd, p)
	}
	result, err := ctrl.FileReader.Read(p)
	if err != nil {
		return err
	}
	if prog, err := expr.NewBool(result.Text); err != nil {
		return err
	} else {
		scr.SetBoolProgram(prog)
	}
	return nil
}

func (ctrl Controller) ReadExternalFiles(ctx context.Context, wd string) error { //nolint:gocognit
	for i, phase := range ctrl.Config.Phases {
		for j, task := range phase.Tasks {
			if err := ctrl.readTemplateFile(task.Command.CommandFile, wd, &task.Command.Command); err != nil {
				return err
			}
			if err := ctrl.readTemplateFile(task.Command.StdinFile, wd, &task.Command.Stdin); err != nil {
				return err
			}
			for k, v := range task.Command.Env.Vars {
				if v.ValueFile != "" {
					p := v.ValueFile
					if !filepath.IsAbs(p) {
						p = filepath.Join(wd, p)
					}
					result, err := ctrl.FileReader.Read(p)
					if err != nil {
						return err
					}
					tpl, err := template.Compile(result.Text)
					if err != nil {
						return err
					}
					v.Value = tpl
				}
				task.Command.Env.Vars[k] = v
			}
			if task.WriteFile.TemplateFile != "" {
				if err := ctrl.readTemplateFile(task.WriteFile.TemplateFile, wd, &task.WriteFile.Template); err != nil {
					return err
				}
			}
			if err := ctrl.readScript(task.InputFile, wd, &task.Input); err != nil {
				return err
			}
			if err := ctrl.readScript(task.OutputFile, wd, &task.Output); err != nil {
				return err
			}
			if err := ctrl.readBoolScript(task.WhenFile, wd, &task.When); err != nil {
				return err
			}
			phase.Tasks[j] = task
		}
		ctrl.Config.Phases[i] = phase
	}
	return nil
}

func (ctrl Controller) Run(ctx context.Context, wd string) error {
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

	if err := ctrl.ReadExternalFiles(ctx, wd); err != nil {
		return err
	}

	params, err := ctrl.getParams(ctx, pr)
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
		phase, err := ctrl.runPhase(ctx, params, i, wd)
		if phase.Error != nil {
			phase.Status = constant.Failed
		}
		phase.outputResult(ctrl.Stderr, phaseCfg.Name)
		params.Phases[phaseCfg.Name] = phase
		if err != nil {
			return err
		}
		if phase.Exit {
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
