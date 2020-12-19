package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

func GuildCreateHandler(client *redis.Client) func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {

	}
}
