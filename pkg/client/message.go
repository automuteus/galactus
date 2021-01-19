package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusClient) SendChannelMessage(channelID string, message string) (*discordgo.Message, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.SendMessagePartial, channelID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return nil, err
	}
	var msg discordgo.Message
	err = json.Unmarshal(respBytes, &msg)
	return &msg, err
}

func (galactus *GalactusClient) SendChannelMessageEmbed(channelID string, embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	message, err := json.Marshal(*embed)
	if err != nil {
		return nil, err
	}

	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.SendMessageEmbedPartial, channelID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return nil, err
	}
	var msg discordgo.Message
	err = json.Unmarshal(respBytes, &msg)
	return &msg, err
}

func (galactus *GalactusClient) EditChannelMessageEmbed(channelID, messageID string, embed discordgo.MessageEmbed) (*discordgo.Message, error) {
	message, err := json.Marshal(embed)
	if err != nil {
		return nil, err
	}

	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.EditMessageEmbedPartial, channelID, messageID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return nil, err
	}
	var msg discordgo.Message
	err = json.Unmarshal(respBytes, &msg)
	return &msg, err
}

func (galactus *GalactusClient) DeleteChannelMessage(channelID, messageID string) error {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.DeleteMessagePartial, channelID, messageID)
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
	return nil
}
