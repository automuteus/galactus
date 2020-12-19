package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

func GuildDeleteHandler(client *redis.Client) func(s *discordgo.Session, m *discordgo.GuildDelete) {
	return func(s *discordgo.Session, m *discordgo.GuildDelete) {

	}
}
