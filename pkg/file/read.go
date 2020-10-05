package file

import (
	"io/ioutil"
	"os"

	"github.com/suzuki-shunsuke/buildflow/pkg/domain"
)

type (
	Reader struct{}
	Writer struct{}
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

func (writer Writer) openWriteFile(path string) (*os.File, error) {
	if path == "" {
		return ioutil.TempFile("", "")
	}
	return os.Create(path)
}

func (writer Writer) Write(path, text string) (domain.FileResult, error) {
	b := []byte(text)
	f, err := writer.openWriteFile(path)
	if err != nil {
		return domain.FileResult{}, err
	}
	defer f.Close()

	if _, err := f.Write(b); err != nil {
		return domain.FileResult{}, err
	}

	stat, err := f.Stat()
	if err != nil {
		return domain.FileResult{}, err
	}

	return domain.FileResult{
		Text:    text,
		Path:    f.Name(),
		ModTime: stat.ModTime(),
		Size:    stat.Size(),
		Mode:    stat.Mode(),
		IsDir:   stat.IsDir(),
	}, nil
}
