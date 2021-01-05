package redis

import (
	"context"
	"errors"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

const SnowflakeLockDuration = time.Second * 3

func LockSnowflake(ctx context.Context, client *redis.Client, snowflake string) *redislock.Lock {
	locker := redislock.New(client)
	lock, err := locker.Obtain(ctx, rediskey.SnowflakeLockID(snowflake), SnowflakeLockDuration, nil)
	if errors.Is(err, redislock.ErrNotObtained) {
		return nil
	} else if err != nil {
		log.Println(err)
		return nil
	}
	return lock
}

func IsUserBanned(client *redis.Client, userID string) bool {
	v, err := client.Exists(context.Background(), rediskey.UserSoftban(userID)).Result()
	if err != nil {
		log.Println(err)
		return false
	}
	return v == 1 // = 1 means the user is present, and thus rate-limited
}
