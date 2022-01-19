package redis

import (
	"github.com/go-redsync/redsync/v4"
)

// default lock is 8 seconds; only allow 4 retries * 500ms interval for 2secs of leniency without duplicate processing
func LockSnowflake(locker *redsync.Redsync, snowflake string) (*redsync.Mutex, error) {
	mutex := locker.NewMutex(snowflake, redsync.WithTries(4))
	err := mutex.Lock()
	if err != nil {
		return nil, err
	}
	return mutex, nil
}
