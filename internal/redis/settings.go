package redis

import (
	"context"
	"encoding/json"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/settings"
	"github.com/go-redis/redis/v8"
	"time"
)

func GetPrefixFromRedis(client *redis.Client, guildID string) (string, error) {
	key := rediskey.GuildPrefix(rediskey.HashGuildID(guildID))
	str, err := client.Get(context.Background(), key).Result()
	if err != nil {
		sett, err := GetSettingsFromRedis(client, guildID)
		if err != nil {
			return "", err
		}
		client.Set(context.Background(), key, sett.CommandPrefix, time.Hour*12)
		return sett.CommandPrefix, err
	}
	return str, err
}

func GetSettingsFromRedis(client *redis.Client, guildID string) (*settings.GuildSettings, error) {
	var sett settings.GuildSettings
	key := rediskey.GuildSettings(rediskey.HashGuildID(guildID))

	str, err := client.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal([]byte(str), &sett)
	if err != nil {
		return nil, err
	}
	return &sett, nil
}
