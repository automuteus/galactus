package shard_manager

import (
	"errors"
	"github.com/automuteus/galactus/internal/handler"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/jonas747/dshardmanager"
	"go.uber.org/zap"
	"math/rand"
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

func AddHandlers(logger *zap.Logger, manager *dshardmanager.Manager, client *redis.Client) {
	manager.AddHandler(handler.GuildCreateHandler(logger, client))
	manager.AddHandler(handler.GuildDeleteHandler(logger, client))

	manager.AddHandler(handler.VoiceStateUpdateHandler(logger, client))
	manager.AddHandler(handler.MessageCreateHandler(logger, client))
	manager.AddHandler(handler.MessageReactionAddHandler(logger, client))

	manager.AddHandler(handler.RateLimitHandler(logger, client))
}

const MaxInvalidRandomSessions = 5

func GetRandomSession(manager *dshardmanager.Manager) (*discordgo.Session, error) {
	max := manager.GetNumShards()
	sess := manager.Session(rand.Intn(max))
	i := 1

	for sess == nil {
		if i > MaxInvalidRandomSessions {
			return nil, errors.New("exceeded maximum retries for random session")
		}
		i++
		r := rand.Intn(max)
		sess = manager.Session(r)
	}
	return sess, nil
}
