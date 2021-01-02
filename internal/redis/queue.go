package redis

import (
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
)

const GatewayMessageKey = "automuteus:gateway:message"

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
	Data        string
}

func PushDiscordMessage(client *redis.Client, messageType DiscordMessageType, data []byte) error {
	s := DiscordMessage{
		MessageType: messageType,
		Data:        string(data),
	}
	byt, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return client.LPush(context.Background(), GatewayMessageKey, byt).Err()
}

func PopRawDiscordMessage(client *redis.Client) (string, error) {
	res, err := client.RPop(context.Background(), GatewayMessageKey).Result()
	if err != nil {
		return "", err
	}

	return res, nil
}

func DiscordMessagesSize(client *redis.Client) (int64, error) {
	return client.LLen(context.Background(), GatewayMessageKey).Result()
}
