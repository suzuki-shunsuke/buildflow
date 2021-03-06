package examples_test

import (
	"testing"

	"gotest.tools/v3/icmd"
)

func TestBuildflow(t *testing.T) { //nolint:funlen
	data := []struct {
		title string
		file  string
		exp   icmd.Expected
	}{
		{
			title: "hello world",
			file:  "hello_world.yaml",
		},
		{
			title: "define meta parameters",
			file:  "meta.yaml",
		},
		{
			title: "read command from a file (task.command_file)",
			file:  "command_file.yaml",
		},
		{
			title: "run tasks in parallel",
			file:  "parallel.yaml",
		},
		{
			title: "the task bar depends on the task foo",
			file:  "task_dependency.yaml",
		},
		{
			title: "task.when",
			file:  "task_when.yaml",
		},
		{
			title: "read_file",
			file:  "read_file.yaml",
		},
		{
			title: "write_file",
			file:  "write_file.yaml",
		},
		{
			title: "command's standard input",
			file:  "stdin.yaml",
		},
		{
			title: "import phases from a file",
			file:  "import_phases.yaml",
		},
		{
			title: "import tasks from a file",
			file:  "import_tasks.yaml",
		},
		{
			title: "dynamic task by items",
			file:  "dynamic_task.yaml",
		},
		{
			title: "skip a phase",
			file:  "skip_phase.yaml",
		},
		{
			title: "buildflow run fails as expected",
			file:  "fail.yaml",
			exp: icmd.Expected{
				ExitCode: 1,
			},
		},
		{
			title: "if there are unknown fields in configuration file, buildflow run fails",
			file:  "unknown_field.yaml",
			exp: icmd.Expected{
				ExitCode: 1,
			},
		},
	}
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			result := icmd.RunCmd(icmd.Command("buildflow", "run", "-c", d.file))
			result.Assert(t, d.exp)
		})
	}
}
