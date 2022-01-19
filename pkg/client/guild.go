package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/premium"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusClient) GetGuild(guildID string) (*discordgo.Guild, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildPartial, guildID)
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

	var guild discordgo.Guild
	err = json.Unmarshal(respBytes, &guild)
	if err != nil {
		return nil, err
	}
	return &guild, nil
}

func (galactus *GalactusClient) GetGuildChannels(guildID string) ([]*discordgo.Channel, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildChannelsPartial, guildID)
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

	var channels []*discordgo.Channel
	err = json.Unmarshal(respBytes, &channels)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

func (galactus *GalactusClient) GetGuildEmojis(guildID string) ([]*discordgo.Emoji, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildEmojisPartial, guildID)
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

	var emojis []*discordgo.Emoji
	err = json.Unmarshal(respBytes, &emojis)
	if err != nil {
		return nil, err
	}
	return emojis, nil
}

func (galactus *GalactusClient) CreateGuildEmoji(guildID, emojiName, content string) (*discordgo.Emoji, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.CreateGuildEmojiPartial, guildID, emojiName)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(content))
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

	var emoji discordgo.Emoji
	err = json.Unmarshal(respBytes, &emoji)
	if err != nil {
		return nil, err
	}
	return &emoji, nil
}

func (galactus *GalactusClient) GetGuildMember(guildID, userID string) (*discordgo.Member, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildMemberPartial, guildID, userID)
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

	var member discordgo.Member
	err = json.Unmarshal(respBytes, &member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (galactus *GalactusClient) GetGuildRoles(guildID string) ([]*discordgo.Role, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildRolesPartial, guildID)
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

	var roles []*discordgo.Role
	err = json.Unmarshal(respBytes, &roles)
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func (galactus *GalactusClient) GetGuildPremium(guildID string) (*premium.PremiumRecord, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.GetGuildPremiumPartial, guildID)
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
	var rec premium.PremiumRecord
	err = json.Unmarshal(respBytes, &rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
