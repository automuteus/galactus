package validate

import (
	"errors"
	"strconv"
)

const DiscordEpoch = 1420070400000

func ValidSnowflake(snowflake string) (bool, error) {
	if snowflake == "" {
		return false, errors.New("empty string")
	}

	num, err := strconv.ParseUint(snowflake, 10, 64)
	if err != nil {
		return false, err
	}

	if num < DiscordEpoch {
		return false, errors.New("too small (prior to discord epoch)")
	}

	return true, nil
}
