package examples_test

import (
	"testing"

	"gotest.tools/v3/icmd"
)

func TestHelloWorld(t *testing.T) {
	result := icmd.RunCmd(icmd.Command("buildflow", "run", "-c", "hello_world.yaml"))
	result.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      "",
	})
}

func TestParallel(t *testing.T) {
	result := icmd.RunCmd(icmd.Command("buildflow", "run", "-c", "parallel.yaml"))
	result.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      "",
	})
}
