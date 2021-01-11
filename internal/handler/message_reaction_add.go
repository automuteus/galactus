package handler

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func MessageReactionAddHandler(logger *zap.Logger, client *redis.Client) func(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
		if m == nil {
			return
		}

		// ignore reactions from the bot
		if m.UserID == s.State.User.ID {
			return
		}

		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for MessageReactionAdd message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, discord_message.MessageReactionAdd, byt)
		if err != nil {
			logger.Error("error pushing to Redis for MessageReactionAdd message",
				zap.Error(err))
		} else {
			LogDiscordMessagePush(logger, discord_message.MessageReactionAdd, m.GuildID, m.ChannelID, m.UserID, m.MessageID)
		}
	}
}
