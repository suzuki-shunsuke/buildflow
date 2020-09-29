package domain

import (
	"net/http"
	"time"
)

type Result struct {
	// queue
	// failed
	// succeeded
	// running
	// skipped
	// cancelled
	Status  string
	Output  map[string]interface{}
	Time    Time
	Type    string
	Command CommandResult
	File    FileResult
	HTTP    HTTPResult
	Error   error
}

func (result Result) IsFinished() bool {
	return !(result.Status == "running" || result.Status == "queue")
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
	Text string
}

type HTTPResult struct {
	Text     string
	Response *http.Response
}
