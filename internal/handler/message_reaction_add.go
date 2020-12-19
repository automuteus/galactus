package handler

import (
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
)

func MessageReactionAddHandler(client *redis.Client) func(s *discordgo.Session, m *discordgo.MessageReactionAdd) {
	return func(s *discordgo.Session, m *discordgo.MessageReactionAdd) {

	}
}
