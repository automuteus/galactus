package redis

import (
	"context"
	"encoding/json"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/go-redis/redis/v8"
)

const GatewayMessageKey = "automuteus:gateway:message"

func PushDiscordMessage(client *redis.Client, messageType discord_message.DiscordMessageType, data []byte) error {
	s := discord_message.DiscordMessage{
		MessageType: messageType,
		Data:        data,
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
