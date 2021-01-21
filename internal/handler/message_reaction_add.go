package handler

import (
	"context"
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/automuteus/utils/pkg/rediskey"
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

		// if no active games in this text channel, completely ignore this message reaction message
		res, err := client.Exists(context.Background(), rediskey.TextChannelPtr(m.GuildID, m.ChannelID)).Result()
		if err != nil || res == 0 {
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
			logger.Info("pushed discord message to Redis",
				zap.String("type", discord_message.DiscordMessageTypeStrings[discord_message.MessageReactionAdd]),
				zap.String("guild_id", m.GuildID),
				zap.String("channel_id", m.ChannelID),
				zap.String("user_id", m.UserID),
				zap.String("id", m.MessageID),
			)
		}
	}
}
