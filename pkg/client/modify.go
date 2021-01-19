package galactus_client

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/discord"
	"github.com/bsm/redislock"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusClient) ModifyUsers(guildID, connectCode string, request discord.UserModifyRequest, lock *redislock.Lock) *discord.MuteDeafenSuccessCounts {
	if lock != nil {
		defer lock.Release(context.Background())
	}
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.ModifyUserPartial, guildID, connectCode)
	jBytes, err := json.Marshal(request)
	if err != nil {
		return nil
	}

	resp, err := galactus.client.Post(url, "application/json", bytes.NewBuffer(jBytes))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	mds := discord.MuteDeafenSuccessCounts{}
	jBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return &mds
	}
	err = json.Unmarshal(jBytes, &mds)
	if err != nil {
		galactus.logger.Error("error unmarshalling response body",
			zap.Error(err),
			zap.String("url", url),
		)
		return &mds
	}
	return &mds
}
