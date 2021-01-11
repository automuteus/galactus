package discord_message

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
