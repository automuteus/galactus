package handler

import (
	"context"
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func MessageCreateHandler(logger *zap.Logger, client *redis.Client) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m == nil {
			return
		}
		// ignore messages created by the bot
		if m.Author == nil || m.Author.ID == s.State.User.ID {
			return
		}

		// TODO should find an efficient way to hook into a guild's prefix here. Would allow for filtering messages
		// quickly without pushing them into the queue

		// TODO softban the users at this level; bot logic shouldn't have to worry about it

		if redis_utils.IsUserBanned(client, m.Author.ID) {
			logger.Info("ignoring message from softbanned user",
				zap.String("author ID", m.Author.ID),
				zap.String("message ID", m.Message.ID),
				zap.String("contents", m.Message.Content))
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
			LogDiscordMessagePush(logger, discord_message.MessageCreate, m.GuildID, m.ChannelID, m.Author.ID, m.ID)
		}
	}
}
