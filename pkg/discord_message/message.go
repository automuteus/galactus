package discord_message

import (
	"github.com/bwmarrin/discordgo"
	"time"
)

type DiscordMessageType int

const (
	GuildCreate DiscordMessageType = iota
	GuildDelete
	VoiceStateUpdate
	MessageCreate
	MessageReactionAdd
)

var DiscordMessageTypeStrings = []string{
	"GuildCreate",
	"GuildDelete",
	"VoiceStateUpdate",
	"MessageCreate",
	"MessageReactionAdd",
}

type DiscordMessage struct {
	MessageType DiscordMessageType
	Data        []byte
}

func MentionByUserID(userID string) string {
	return "<@!" + userID + ">"
}

func DeleteMessageWorker(sess *discordgo.Session, channelID, messageID string, wait time.Duration) error {
	time.Sleep(wait)

	return sess.ChannelMessageDelete(channelID, messageID)
}
