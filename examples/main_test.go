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
			exp: icmd.Expected{
				ExitCode: 0,
				Err:      "",
			},
		},
		{
			title: "run tasks in parallel",
			file:  "parallel.yaml",
			exp: icmd.Expected{
				ExitCode: 0,
				Err:      "",
			},
		},
		{
			title: "the task bar depends on the task foo",
			file:  "task_dependency.yaml",
			exp: icmd.Expected{
				ExitCode: 0,
				Err:      "",
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
