package domain

import (
	"net/http"
	"os"
	"time"

	"github.com/suzuki-shunsuke/buildflow/pkg/constant"
)

type Result struct {
	// queue
	// failed
	// succeeded
	// running
	// skipped
	// cancelled
	Status  string
	Input   interface{}
	Output  interface{}
	Time    Time
	Type    string
	Command CommandResult
	File    FileResult
	HTTP    HTTPResult
	Error   error
}

type PhaseResult struct {
	// succeeded
	// failed
	// skipped
	Status string
	Time   Time
	Error  error
	Output map[string]interface{}
	Tasks  []Result
}

func (result Result) IsFinished() bool {
	return !(result.Status == constant.Running || result.Status == constant.Queue)
}

type Time struct {
	Start time.Time
	End   time.Time
}

type CommandResult struct {
	ExitCode       int
	Cmd            string
	Stdout         string
	Stderr         string
	CombinedOutput string
}

type FileResult struct {
	Text    string
	Path    string
	ModTime time.Time
	Size    int64
	Mode    os.FileMode
	IsDir   bool
}

func (fileResult FileResult) ToTemplate() map[string]interface{} {
	return map[string]interface{}{
		"Text":    fileResult.Text,
		"Mode":    fileResult.Mode.String(),
		"Path":    fileResult.Path,
		"ModTime": fileResult.ModTime,
		"Size":    fileResult.Size,
		"IsDir":   fileResult.IsDir,
	}
}

type HTTPResult struct {
	Text     string
	Response *http.Response
}
