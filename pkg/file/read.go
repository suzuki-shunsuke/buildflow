package file

import (
	"io/ioutil"

	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
)

type (
	Reader struct{}
)

func (reader Reader) Read(path string) (domain.FileResult, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return domain.FileResult{}, err
	}
	return domain.FileResult{
		Text: string(b),
	}, nil
}
