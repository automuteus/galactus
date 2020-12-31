package handler

import (
	"github.com/automuteus/galactus/internal/redis"
	"go.uber.org/zap"
)

func LogDiscordMessagePush(logger *zap.Logger, msgType redis.DiscordMessageType, guildID, channelID, userID, ID string) {
	logger.Info("pushed discord message to Redis",
		zap.String("type", redis.DiscordMessageTypeStrings[msgType]),
		zap.String("guild_id", guildID),
		zap.String("channel_id", channelID),
		zap.String("user_id", userID),
		zap.String("id", ID),
	)
}
