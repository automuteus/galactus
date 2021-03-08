package validate

import (
	"errors"
	"fmt"
	"github.com/automuteus/utils/pkg/capture"
	"github.com/automuteus/utils/pkg/discord"
)

const ConnectCodeLength = 8

func ValidConnectCode(code string) (bool, error) {
	if code == "" {
		return false, errors.New("empty connect code")
	}

	if len(code) != ConnectCodeLength {
		return false, errors.New(fmt.Sprintf("length of code is %d, not the expected %d", len(code), ConnectCodeLength))
	}
	return true, nil
}

func ValidTaskID(taskID string) (bool, error) {
	if taskID == "" {
		return false, errors.New("empty taskID")
	}

	if len(taskID) != discord.IDLength {
		return false, errors.New(fmt.Sprintf("length of code is %d, not the expected %d", len(taskID), discord.IDLength))
	}
	return true, nil
}

func ValidEventType(eventType int) (bool, error) {
	if eventType == int(capture.Connection) || eventType == int(capture.Lobby) || eventType == int(capture.State) ||
		eventType == int(capture.Player) || eventType == int(capture.GameOver) {
		return true, nil
	}
	return false, errors.New("eventType is not a valid value")
}
