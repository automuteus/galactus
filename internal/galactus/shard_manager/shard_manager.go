package shard_manager

import (
	"github.com/automuteus/galactus/internal/handler"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/jonas747/dshardmanager"
	"go.uber.org/zap"
)

func MakeShardManager(logger *zap.Logger, token string, intent *discordgo.Intent) *dshardmanager.Manager {
	manager := dshardmanager.New("Bot " + token)
	manager.Name = "AutoMuteUs"

	recommended, err := manager.GetRecommendedCount()
	if err != nil {
		logger.Fatal("failed to obtain recommended shard count",
			zap.Error(err))
	}

	manager.SetNumShards(recommended)

	logger.Info("starting shard manager",
		zap.Int("num shards", manager.GetNumShards()))

	err = manager.Start()
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

	return manager
}

func AddHandlers(logger *zap.Logger, manager *dshardmanager.Manager, client *redis.Client, botPrefix string) {
	manager.AddHandler(handler.GuildCreateHandler(logger, client))
	manager.AddHandler(handler.GuildDeleteHandler(logger, client))

	manager.AddHandler(handler.VoiceStateUpdateHandler(logger, client))
	manager.AddHandler(handler.MessageCreateHandler(logger, client, botPrefix))
	manager.AddHandler(handler.MessageReactionAddHandler(logger, client))
}

func AddRateLimitHandler(manager *dshardmanager.Manager, handler func(sess *discordgo.Session, rl *discordgo.RateLimit)) {
	manager.AddHandler(handler)
}
