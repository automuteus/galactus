package galactus_client

import (
	"bytes"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusClient) AddReaction(channelID, messageID, emojiID string) error {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.AddReactionPartial, channelID, messageID, emojiID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return err
	}

	return err
}

func (galactus *GalactusClient) RemoveReaction(channelID, messageID, emojiID, userID string) error {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.RemoveReactionPartial, channelID, messageID, emojiID, userID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return err
	}

	return err
}

func (galactus *GalactusClient) RemoveAllReactions(channelID, messageID string) error {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.RemoveAllReactionsPartial, channelID, messageID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return err
	}

	return err
}
