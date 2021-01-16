package validate

import (
	"errors"
	"fmt"
	"github.com/automuteus/utils/pkg/task"
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

	if len(taskID) != task.IDLength {
		return false, errors.New(fmt.Sprintf("length of code is %d, not the expected %d", len(taskID), task.IDLength))
	}
	return true, nil
}
