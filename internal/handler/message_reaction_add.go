package handler

import (
	"context"
	"encoding/json"
	"fmt"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"time"
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

		// TODO how to easily and cleanly localize these messages?
		if redis_utils.IsUserRateLimitedGeneral(client, m.UserID) {
			// record the violation with this call
			if redis_utils.IncrementRateLimitExceed(client, m.UserID) {
				msg, err := s.ChannelMessageSend(m.ChannelID,
					fmt.Sprintf("%s has been spamming. I'm ignoring them for the next %f minutes.",
						discord_message.MentionByUserID(m.UserID),
						redis_utils.SoftbanDuration.Minutes()))
				if err != nil {
					logger.Error("error posting ratelimit ban message",
						zap.Error(err),
					)
				} else {
					go discord_message.DeleteMessageWorker(s, msg.ChannelID, msg.ID, time.Second*3)
				}
				return
			} else {
				msg, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you're reacting too fast! Please slow down!", discord_message.MentionByUserID(m.UserID)))
				if err != nil {
					logger.Error("error posting ratelimit warning message",
						zap.Error(err),
					)
				} else {
					go discord_message.DeleteMessageWorker(s, msg.ChannelID, msg.ID, time.Second*3)
				}
				return
			}
		}
		redis_utils.MarkUserRateLimit(client, m.UserID, "", 0)

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
