package handler

import (
	"context"
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"strings"
)

func MessageCreateHandler(logger *zap.Logger, client *redis.Client, globalPrefix string) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m == nil {
			return
		}
		// ignore messages created by the bot
		if m.Author == nil || m.Author.ID == s.State.User.ID {
			return
		}

		snowflakeLock := redis_utils.LockSnowflake(context.Background(), client, m.ID)
		// couldn't obtain lock; bail bail bail!
		if snowflakeLock == nil {
			logger.Info("could not obtain snowflake lock",
				zap.String("type", "MessageCreate"),
				zap.Int("shard ID", s.ShardID),
				zap.String("snowflakeID", m.ID))
			return
		}
		defer snowflakeLock.Release(context.Background())

		if redis_utils.IsUserBanned(client, m.Author.ID) {
			logger.Info("ignoring message from softbanned user",
				zap.String("author ID", m.Author.ID),
				zap.String("message ID", m.Message.ID),
				zap.String("contents", m.Message.Content))
			return
		}

		detectedPrefix := ""
		sett, err := redis_utils.GetSettingsFromRedis(client, m.GuildID)

		if sett != nil && err == nil {
			if strings.HasPrefix(m.Content, sett.CommandPrefix) {
				detectedPrefix = sett.CommandPrefix
			}
		}

		if detectedPrefix == "" {
			if strings.HasPrefix(m.Content, "<@!"+s.State.User.ID+">") {
				detectedPrefix = "<@!" + s.State.User.ID + ">"
			} else if strings.HasPrefix(m.Content, globalPrefix) {
				detectedPrefix = globalPrefix
			}
		}

		// wasn't a message for the bot; don't push to redis
		if detectedPrefix == "" {
			return
		}

		m.Content = stripPrefix(m.Content, detectedPrefix)

		// TODO softban the users at this level; bot logic shouldn't have to worry about it

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
	newMsg := strings.Replace(msg, prefix+"", "", 1)
	// didn't substitute anything
	if len(newMsg) == len(msg) {
		return strings.Replace(msg, prefix, "", 1)
	}
	return newMsg
}
