package constant

import "errors"

const (
	Command = "command"
	File    = "file"
	HTTP    = "http"

	Failed    = "failed"
	Succeeded = "succeeded"
	Running   = "running"
	Skipped   = "skipped"
	Queue     = "queue"
)

// result
const Result = "result"

var ErrNoBoolVariable = errors.New(`the variable "result" isn't defined`)
