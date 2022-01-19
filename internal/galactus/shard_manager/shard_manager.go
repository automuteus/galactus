package shard_manager

import (
	"github.com/automuteus/galactus/internal/handler"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/jonas747/dshardmanager"
	"go.uber.org/zap"
)

func MakeShardManager(logger *zap.Logger, token string, numShards int) *dshardmanager.Manager {
	manager := dshardmanager.New("Bot " + token)
	manager.Name = "AutoMuteUs"

	recommended, err := manager.GetRecommendedCount()
	if err != nil {
		logger.Fatal("failed to obtain recommended shard count",
			zap.Error(err))
	}
	if numShards > 0 {
		logger.Info("obtained recommended number of shards, but using provided value instead",
			zap.Int("recommended", recommended),
			zap.Int("NUM_SHARDS", numShards),
		)
		manager.SetNumShards(numShards)
	} else {
		manager.SetNumShards(recommended)
	}

	return manager
}

func AddHandlers(logger *zap.Logger, manager *dshardmanager.Manager, client *redis.Client, botPrefix string) {
	pool := goredis.NewPool(client)

	locker := redsync.New(pool)
	manager.AddHandler(handler.GuildCreateHandler(logger, client, locker))
	manager.AddHandler(handler.GuildDeleteHandler(logger, client, locker))

	manager.AddHandler(handler.VoiceStateUpdateHandler(logger, client, locker))
	manager.AddHandler(handler.MessageCreateHandler(logger, client, locker, botPrefix))
	manager.AddHandler(handler.MessageReactionAddHandler(logger, client, locker))
}

func AddRateLimitHandler(manager *dshardmanager.Manager, handler func(sess *discordgo.Session, rl *discordgo.RateLimit)) {
	manager.AddHandler(handler)
}

func Start(logger *zap.Logger, manager *dshardmanager.Manager, intent discordgo.Intent) {
	logger.Info("starting shard manager",
		zap.Int("num shards", manager.GetNumShards()))

	err := manager.Start()
	if err != nil {
		logger.Fatal("failed to start shard manager",
			zap.Error(err))
	}

	logger.Info("shard manager started successfully")

	manager.Lock()
	for _, v := range manager.Sessions {
		v.Identify.Intents = intent
	}
	manager.Unlock()
}
