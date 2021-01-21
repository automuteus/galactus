package handler

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func GuildDeleteHandler(logger *zap.Logger, client *redis.Client) func(s *discordgo.Session, m *discordgo.GuildDelete) {
	return func(s *discordgo.Session, m *discordgo.GuildDelete) {
		if m == nil {
			return
		}
		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for GuildDelete message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, discord_message.GuildDelete, byt)
		if err != nil {
			logger.Error("error pushing to Redis for GuildDelete message",
				zap.Error(err))
		} else {
			logger.Info("pushed discord message to Redis",
				zap.String("type", discord_message.DiscordMessageTypeStrings[discord_message.GuildDelete]),
				zap.String("guild_id", m.ID),
				zap.String("user_id", m.OwnerID),
				zap.String("id", m.ID),
			)
		}
	}
}
