package redis

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/automuteus/utils/pkg/discord"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/go-redis/redis/v8"
	"time"
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

func PopRawDiscordMessageTimeout(client *redis.Client, timeout time.Duration) (string, error) {
	res, err := client.BRPop(context.Background(), timeout, GatewayMessageKey).Result()
	if err != nil {
		return "", err
	} else if len(res) < 2 { // we expect length 2+ because BRPOP returns the key and the value
		return "", errors.New("empty queue")
	}
	return res[1], nil
}

func PushCaptureClientTask(client *redis.Client, connectCode string, taskObj discord.ModifyTask, expiry time.Duration) error {
	jBytes, err := json.Marshal(taskObj)
	if err != nil {
		return err
	}
	err = client.LPush(context.Background(), rediskey.TasksList(connectCode), jBytes).Err()
	if err != nil {
		return err
	}
	return client.Expire(context.Background(), rediskey.TasksList(connectCode), expiry).Err()
}

func PopRawCaptureClientTask(client *redis.Client, connectCode string, timeout time.Duration) (string, error) {
	res, err := client.BRPop(context.Background(), timeout, rediskey.TasksList(connectCode)).Result()
	if err != nil {
		return "", err
	} else if len(res) < 2 { // we expect length 2+ because BRPOP returns the key and the value
		return "", errors.New("empty queue")
	}
	return res[1], nil
}

func DiscordMessagesSize(client *redis.Client) (int64, error) {
	return client.LLen(context.Background(), GatewayMessageKey).Result()
}
