package examples_test

import (
	"testing"

	"gotest.tools/v3/icmd"
)

func TestBuildflow(t *testing.T) {
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
			title: "run tasks in parallel",
			file:  "parallel.yaml",
		},
		{
			title: "the task bar depends on the task foo",
			file:  "task_dependency.yaml",
		},
		{
			title: "task.when is true",
			file:  "task_when_true.yaml",
		},
		{
			title: "read_file",
			file:  "read_file.yaml",
		},
		{
			title: "command's standard input",
			file:  "stdin.yaml",
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
