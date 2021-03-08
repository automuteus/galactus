package redis

import (
	"context"
	"fmt"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/go-redis/redis/v8"
	"time"
)

// 15 minute timeout
const GameTimeoutSeconds = 900

// only deletes from the guild's responsibility, NOT the entire guild counter!
func AnyActiveGamesInGuild(client *redis.Client, guildID string) bool {
	hash := rediskey.ActiveGamesForGuild(guildID)

	games, err := client.ZCard(context.Background(), hash).Result()

	if err != nil {
		return false
	}

	return games > 0
}

func PurgeOldGuildGames(client *redis.Client, guildID string) {
	hash := rediskey.ActiveGamesForGuild(guildID)

	before := time.Now().Add(-time.Second * GameTimeoutSeconds).Unix()

	client.ZRemRangeByScore(context.Background(), hash, "-inf", fmt.Sprintf("%d", before))
}
