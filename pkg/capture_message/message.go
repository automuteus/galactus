package capture_message

import "github.com/automuteus/utils/pkg/task"

type CaptureMessage struct {
	MessageType task.JobType
	Data        []byte
}
