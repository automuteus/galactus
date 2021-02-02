package redis

import (
	"github.com/go-redsync/redsync/v4"
)

func LockSnowflake(locker *redsync.Redsync, snowflake string) (*redsync.Mutex, error) {
	mutex := locker.NewMutex(snowflake)
	err := mutex.Lock()
	if err != nil {
		return nil, err
	}
	return mutex, nil
}
