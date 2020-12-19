package handler

import (
	"context"
	redis2 "github.com/automuteus/galactus/internal/redis"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

func MessageCreateHandler(client *redis.Client) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// ignore messages created by the bot
		if m.Author.ID == s.State.User.ID {
			return
		}

		if redis2.IsUserBanned(client, m.Author.ID) {
			return
		}

		snowflakeLock := redis2.LockSnowflake(context.Background(), client, m.ID)
		// couldn't obtain lock; bail bail bail!
		if snowflakeLock == nil {
			return
		}
		defer snowflakeLock.Release(context.Background())
	}
}
