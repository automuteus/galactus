package redis

import (
	"github.com/go-redsync/redsync/v4"
)

// default lock is 8 seconds; only allow 1 consumer to pick up the message (no retries in case it's processed/released quickly)
func LockSnowflake(locker *redsync.Redsync, snowflake string) (*redsync.Mutex, error) {
	mutex := locker.NewMutex(snowflake, redsync.WithTries(1))
	err := mutex.Lock()
	if err != nil {
		return nil, err
	}
	return mutex, nil
}
