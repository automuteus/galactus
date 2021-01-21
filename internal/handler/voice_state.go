package handler

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func VoiceStateUpdateHandler(logger *zap.Logger, client *redis.Client) func(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
	return func(s *discordgo.Session, m *discordgo.VoiceStateUpdate) {
		// TODO filter voice changes when a game isn't happening in this guild.
		// probably won't work to filter changes just by the voice channel ID; results in ppl not being unmuted when they leave the VC

		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for VoiceStateUpdate message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, discord_message.VoiceStateUpdate, byt)
		if err != nil {
			logger.Error("error pushing to Redis for VoiceStateUpdate message",
				zap.Error(err))
		} else {
			logger.Info("pushed discord message to Redis",
				zap.String("type", discord_message.DiscordMessageTypeStrings[discord_message.VoiceStateUpdate]),
				zap.String("guild_id", m.GuildID),
				zap.String("channel_id", m.ChannelID),
				zap.String("user_id", m.UserID),
				zap.String("id", m.SessionID),
			)
		}
	}
}
