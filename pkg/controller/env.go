package controller

import (
	"fmt"

	"github.com/suzuki-shunsuke/buildflow/pkg/config"
)

func renderEnvs(envs config.Envs, params Params) ([]string, error) {
	m := make([]string, len(envs.Vars))
	for i, env := range envs.Vars {
		k, err := env.Key.Render(params.ToTemplate())
		if err != nil {
			return nil, fmt.Errorf(`failed to render env key %d: %w`, i, err)
		}
		v, err := env.Value.Render(params.ToTemplate())
		if err != nil {
			return nil, fmt.Errorf(`failed to render env value %d: %w`, i, err)
		}
		m[i] = k + "=" + v
	}
	return m, nil
}
