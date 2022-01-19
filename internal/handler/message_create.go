package handler

import (
	"encoding/json"
	"fmt"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"go.uber.org/zap"
	"strings"
	"time"
)

func MessageCreateHandler(logger *zap.Logger, client *redis.Client, locker *redsync.Redsync, globalPrefix string) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m == nil {
			return
		}
		// ignore messages created by the bot
		if m.Author == nil || m.Author.ID == s.State.User.ID || m.Author.Bot {
			return
		}

		snowflakeMutex, err := redis_utils.LockSnowflake(locker, m.ID)
		// couldn't obtain lock; bail bail bail!
		if snowflakeMutex == nil {
			//logger.Info("could not obtain snowflake lock",
			//	zap.String("type", "MessageCreate"),
			//	zap.Int("shard ID", s.ShardID),
			//	zap.String("snowflakeID", m.ID))
			return
		}
		// explicitly DO NOT unlock the snowflake! We don't want anyone else processing the event!

		if redis_utils.IsUserBanned(client, m.Author.ID) {
			logger.Info("ignoring message from softbanned user",
				zap.String("author ID", m.Author.ID),
				zap.String("message ID", m.Message.ID),
				zap.String("contents", m.Message.Content))
			return
		}

		detectedPrefix := ""
		if strings.HasPrefix(m.Content, "<@!"+s.State.User.ID+">") {
			detectedPrefix = "<@!" + s.State.User.ID + ">"
		} else if strings.HasPrefix(m.Content, "<@"+s.State.User.ID+">") {
			detectedPrefix = "<@" + s.State.User.ID + ">"
		} else if strings.HasPrefix(m.Content, globalPrefix) {
			detectedPrefix = globalPrefix
		}

		if detectedPrefix == "" {
			prefix, err := redis_utils.GetPrefixFromRedis(client, m.GuildID)
			if prefix != "" && err == nil {
				if strings.HasPrefix(m.Content, prefix) {
					detectedPrefix = prefix
				}
			}
		}

		// wasn't a message for the bot; don't push to redis
		if detectedPrefix == "" {
			return
		}

		// TODO how to easily and cleanly localize these messages?
		if redis_utils.IsUserRateLimitedGeneral(client, m.Author.ID) {
			// record the violation with this call
			if redis_utils.IncrementRateLimitExceed(client, m.Author.ID) {
				msg, err := s.ChannelMessageSend(m.ChannelID,
					fmt.Sprintf("%s has been spamming. I'm ignoring them for the next %d minutes.",
						discord_message.MentionByUserID(m.Author.ID),
						int(redis_utils.SoftbanDuration.Minutes())))
				if err != nil {
					logger.Error("error posting ratelimit ban message",
						zap.Error(err),
					)
				} else {
					go discord_message.DeleteMessageWorker(s, msg.ChannelID, msg.ID, redis_utils.SoftbanDuration)
				}
				return
			} else {
				msg, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you're issuing commands too fast! Please slow down!",
					discord_message.MentionByUserID(m.Author.ID)))
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
		redis_utils.MarkUserRateLimit(client, m.Author.ID, "", 0)

		m.Content = stripPrefix(m.Content, detectedPrefix)

		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for MessageCreate message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, discord_message.MessageCreate, byt)
		if err != nil {
			logger.Error("error pushing discord message to Redis for MessageCreate message",
				zap.Error(err))
		} else {
			logger.Info("pushed discord message to Redis",
				zap.String("type", discord_message.DiscordMessageTypeStrings[discord_message.MessageCreate]),
				zap.String("guild_id", m.GuildID),
				zap.String("channel_id", m.ChannelID),
				zap.String("user_id", m.Author.ID),
				zap.String("id", m.ID),
			)
		}
	}
}

func stripPrefix(msg, prefix string) string {
	newMsg := strings.Replace(msg, prefix+" ", "", 1)
	// didn't substitute anything
	if len(newMsg) == len(msg) {
		return strings.Replace(msg, prefix, "", 1)
	}
	return newMsg
}
