package config

import (
	"errors"
	"os"

	"github.com/suzuki-shunsuke/go-findconfig/findconfig"
	"gopkg.in/yaml.v2"
)

type ExistFile func(string) bool

type Reader struct {
	ExistFile ExistFile
}

func (reader Reader) read(p string) (Config, error) {
	cfg := Config{}
	f, err := os.Open(p)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

var ErrNotFound error = errors.New("configuration file isn't found")

func (reader Reader) FindAndRead(cfgPath, wd string) (Config, error) {
	cfg := Config{}
	if cfgPath == "" {
		p := findconfig.Find(wd, reader.ExistFile, ".buildflow.yml", ".buildflow.yaml")
		if p == "" {
			return cfg, ErrNotFound
		}
		cfgPath = p
	}
	return reader.read(cfgPath)
}
