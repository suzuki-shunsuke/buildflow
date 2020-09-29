package domain

const (
	TaskTypeCommand = "command"
	TaskTypeFile    = "file"
	TaskTypeHTTP    = "http"

	TaskResultFailed    = "failed"
	TaskResultSucceeded = "succeeded"
	TaskResultRunning   = "running"
	TaskResultSkipped   = "skipped"
	TaskResultQueue     = "queue"
)
