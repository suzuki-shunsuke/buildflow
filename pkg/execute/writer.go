package execute

import (
	"io"
	"strings"
	"time"

	"github.com/suzuki-shunsuke/buildflow/pkg/locale"
)

const TimeFormat = "15:04:05MST"

type Writer struct {
	writer io.Writer
	filter func(string) string
}

func NewWriter(writer io.Writer, name string) io.Writer {
	return newWriter(writer, genFilter(name))
}

func newWriter(writer io.Writer, filter func(string) string) io.Writer {
	return Writer{
		writer: writer,
		filter: filter,
	}
}

func (writer Writer) Write(p []byte) (int, error) {
	b := []byte(writer.filter(string(p)))
	n, err := writer.writer.Write(b)
	a := n - (len(b) - len(p))
	if a < 0 {
		return 0, err
	}
	return a, err
}

func genFilter(name string) func(string) string {
	return func(p string) string {
		prefix := time.Now().In(locale.UTC()).Format(TimeFormat) + " | " + name + " | "
		arr := strings.Split(p, "\n")
		for i, line := range arr {
			arr[i] = prefix + line
		}
		return strings.Join(arr, "\n") + "\n"
	}
}
