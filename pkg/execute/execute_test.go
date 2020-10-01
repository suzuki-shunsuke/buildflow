package execute_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/suzuki-shunsuke/buildflow/pkg/execute"
)

func TestExecutor_Run(t *testing.T) {
	data := []struct {
		title  string
		params execute.Params
		isErr  bool
	}{
		{
			title: "dry run",
			params: execute.Params{
				Cmd:     "true",
				DryRun:  true,
				Timeout: execute.Timeout{},
			},
		},
		{
			title: "normal",
			params: execute.Params{
				Cmd:     "true",
				Timeout: execute.Timeout{},
			},
		},
		{
			title: "command is failure",
			isErr: true,
			params: execute.Params{
				Cmd:     "false",
				Timeout: execute.Timeout{},
			},
		},
	}
	ctx := context.Background()
	exc := execute.New()
	for _, d := range data {
		d := d
		t.Run(d.title, func(t *testing.T) {
			_, err := exc.Run(ctx, d.params)
			if err != nil {
				if d.isErr {
					return
				}
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}
