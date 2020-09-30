package controller

import "github.com/suzuki-shunsuke/buildflow/pkg/config"

func renderEnvs(envs config.Envs, params Params) ([]string, error) {
	m := make([]string, len(envs.Vars))
	for i, env := range envs.Vars {
		k, err := env.Key.Render(params)
		if err != nil {
			return nil, err
		}
		v, err := env.Value.Render(params)
		if err != nil {
			return nil, err
		}
		m[i] = k + "=" + v
	}
	return m, nil
}
