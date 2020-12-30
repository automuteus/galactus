package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func RateLimitHandler(logger *zap.Logger, client *redis.Client) func(sess *discordgo.Session, rl *discordgo.RateLimit) {
	return func(sess *discordgo.Session, rl *discordgo.RateLimit) {
		logger.Info("rate limit exceeded",
			zap.String("message", rl.Message),
			zap.String("url", rl.URL),
		)
	}
}
