package discord

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"log"
	"time"
)

//func TasksKey(connectCode string) string {
//	return "automuteus:tasks:code:" + connectCode
//}

func BroadcastTaskAckKey(taskID string) string {
	return fmt.Sprintf("automuteus:tasks:broadcast:ack:%s", taskID)
}

func CompleteTaskAckKey(taskID string) string {
	return fmt.Sprintf("automuteus:tasks:complete:ack:%s", taskID)
}

func TasksSubscribeKey(connectCode string) string {
	return "automuteus:tasks:subscribe:" + connectCode
}

func BotTokenIdentifyKey(token string) string {
	return "automuteus:token:identify" + token
}

func BotTokenIdentifyLockKey(token string) string {
	return "automuteus:token:lock" + token
}

type IdentifyThresholds struct {
	HardWindow    time.Duration
	HardThreshold int64

	SoftWindow    time.Duration
	SoftThreshold int64
}

func MarkIdentifyAndLockForToken(client *redis.Client, token string) {
	//log.Println("Marking IDENTIFY for token: " + token)
	key := BotTokenIdentifyKey(token)
	t := time.Now().Unix()
	_, err := client.ZAdd(context.Background(), key, &redis.Z{
		Score:  float64(t),
		Member: t,
	}).Result()
	if err != nil {
		log.Println(err)
	}

	log.Println("Locking token for 5 seconds")
	err = client.Set(context.Background(), BotTokenIdentifyLockKey(token), "", time.Second*5).Err()
	if err != nil {
		log.Println(err)
	}
}

func WaitForToken(client *redis.Client, token string) {
	for IsTokenLocked(client, token) {
		log.Println("Sleeping for 5 seconds while waiting for token to become available")
		time.Sleep(time.Second * 5)
	}
}

func IsTokenLocked(client *redis.Client, token string) bool {
	v, err := client.Exists(context.Background(), BotTokenIdentifyLockKey(token)).Result()
	if err != nil {
		return false
	}

	return v == 1 //=1 means the key is present, hence locked
}

func IsTokenLockedOut(client *redis.Client, token string, thres IdentifyThresholds) bool {
	t := time.Now()
	count, err := client.ZCount(context.Background(), BotTokenIdentifyKey(token),
		fmt.Sprintf("%d", t.Add(-thres.HardWindow).Unix()),
		fmt.Sprintf("%d", t.Unix())).Result()
	if err != nil {
		log.Println(err)
		return false
	}
	if count > thres.HardThreshold {
		return true
	}

	count, err = client.ZCount(context.Background(), BotTokenIdentifyKey(token),
		fmt.Sprintf("%d", t.Add(-thres.SoftWindow).Unix()),
		fmt.Sprintf("%d", t.Unix())).Result()

	if err != nil {
		log.Println(err)
		return false
	}
	return count > thres.SoftThreshold
}
