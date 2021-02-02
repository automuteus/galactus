package handler

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"go.uber.org/zap"
)

func GuildCreateHandler(logger *zap.Logger, client *redis.Client, locker *redsync.Redsync) func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {
		if m == nil {
			return
		}
		snowflakeMutex, err := redis_utils.LockSnowflake(locker, m.ID+"_create")
		// couldn't obtain lock; bail bail bail!
		if snowflakeMutex == nil {
			logger.Info("could not obtain snowflake lock",
				zap.String("type", "GuildCreate"),
				zap.Int("shard ID", s.ShardID),
				zap.String("snowflakeID", m.ID+"_create"))
			return
		}
		defer snowflakeMutex.Unlock()

		byt, err := json.Marshal(m)
		if err != nil {
			logger.Error("error marshalling json for GuildCreate message",
				zap.Error(err))
		}
		err = redis_utils.PushDiscordMessage(client, discord_message.GuildCreate, byt)
		if err != nil {
			logger.Error("error pushing discord message to Redis for GuildCreate",
				zap.Error(err))
		} else {
			logger.Info("pushed discord message to Redis",
				zap.String("type", discord_message.DiscordMessageTypeStrings[discord_message.GuildCreate]),
				zap.String("guild_id", m.ID),
				zap.String("user_id", m.OwnerID),
				zap.String("id", m.ID),
			)
		}
	}
}
