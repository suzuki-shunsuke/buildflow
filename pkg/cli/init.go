package cli

import (
	"io/ioutil"
	"os"

	"github.com/urfave/cli/v2"
)

const cfgTpl = `---
# Configuration file of buildflow, which is a CLI tool for powerful build pipeline
# https://github.com/suzuki-shunsuke/buildflow
pr: false
parallelism: 1
phases:
- name: main
  tasks:
  - name: hello
    command:
      command: echo hello`

func (runner Runner) initAction(c *cli.Context) error {
	if _, err := os.Stat(".buildflow.yml"); err == nil {
		return nil
	}
	if _, err := os.Stat(".buildflow.yaml"); err == nil {
		return nil
	}
	if err := ioutil.WriteFile(".buildflow.yaml", []byte(cfgTpl), 0o755); err != nil { //nolint:gosec
		return err
	}
	return nil
}
