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

func (galactus *GalactusClient) CreateUserChannel(userID string) (*discordgo.Channel, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.UserChannelCreatePartial, userID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
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

	var channel discordgo.Channel
	err = json.Unmarshal(respBytes, &channel)
	if err != nil {
		return nil, err
	}
	return &channel, nil
}
