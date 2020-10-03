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
