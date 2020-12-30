package handler

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func GuildCreateHandler(logger *zap.Logger, client *redis.Client) func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {
		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for GuildCreate message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, redis_utils.GuildCreate, byt)
		if err != nil {
			logger.Error("error pushing discord message to Redis for GuildCreate",
				zap.Error(err))
		} else {
			logger.Info("received GuildCreate message",
				zap.String("ID", m.ID),
			)
		}
	}
}
