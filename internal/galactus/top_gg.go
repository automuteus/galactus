package galactus

import (
	"errors"
)

func (galactus *GalactusAPI) HasUserVoted(userID string) (bool, error) {
	if galactus.topggClient == nil || galactus.botID == "" {
		return false, errors.New("topgg client or BotID has not been initialized and thus cannot be checked")
	}

	return galactus.topggClient.HasUserVoted(galactus.botID, userID)
}
