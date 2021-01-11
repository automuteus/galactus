package handler

import (
	"github.com/automuteus/galactus/pkg/discord_message"
	"go.uber.org/zap"
)

func LogDiscordMessagePush(logger *zap.Logger, msgType discord_message.DiscordMessageType, guildID, channelID, userID, ID string) {
	logger.Info("pushed discord message to Redis",
		zap.String("type", discord_message.DiscordMessageTypeStrings[msgType]),
		zap.String("guild_id", guildID),
		zap.String("channel_id", channelID),
		zap.String("user_id", userID),
		zap.String("id", ID),
	)
}
