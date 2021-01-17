package capture_message

import "github.com/automuteus/utils/pkg/capture"

type CaptureMessage struct {
	MessageType capture.EventType
	Data        []byte
}
