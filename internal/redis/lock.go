package redis

import (
	"github.com/go-redsync/redsync/v4"
	"time"
)

// locks have 5 second duration, 3 seconds total of retries, retries every 500 ms
func LockSnowflake(locker *redsync.Redsync, snowflake string) (*redsync.Mutex, error) {
	mutex := locker.NewMutex(snowflake, redsync.WithExpiry(time.Second*5), redsync.WithRetryDelay(time.Millisecond*500), redsync.WithTries(6))
	err := mutex.Lock()
	if err != nil {
		return nil, err
	}
	return mutex, nil
}
