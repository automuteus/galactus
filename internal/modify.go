package internal

import (
	"context"
	"encoding/json"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/task"
	"log"
	"time"
)

func (tokenProvider *GalactusAPI) attemptOnSecondaryTokens(guildID, userID string, tokens []string, limit int, request task.UserModify) bool {
	if tokens != nil && limit > 0 {
		sess, hToken := tokenProvider.getAnySession(guildID, tokens, limit)
		if sess != nil {
			err := task.ApplyMuteDeaf(sess, guildID, userID, request.Mute, request.Deaf)
			if err != nil {
				log.Println("Failed to apply mute to player with error:")
				log.Println(err)
			} else {
				log.Printf("Successfully applied mute=%v, deaf=%v to User %d using secondary bot: %s\n", request.Mute, request.Deaf, request.UserID, hToken)
				return true
			}
		} else {
			log.Println("No secondary bot tokens found. Trying other methods")
		}
	} else {
		log.Println("Guild has no access to secondary bot tokens; skipping")
	}
	return false
}

func (tokenProvider *GalactusAPI) attemptOnCaptureBot(guildID, connectCode string, gid uint64, timeout time.Duration, request task.UserModify) bool {
	// this is cheeky, but use the connect code as part of the lock; don't issue too many requests on the capture client w/ this code
	if tokenProvider.IncrAndTestGuildTokenComboLock(guildID, connectCode) {
		// if the secondary token didn't work, then next we try the client-side capture request
		taskObj := task.NewModifyTask(gid, request.UserID, task.PatchParams{
			Deaf: request.Deaf,
			Mute: request.Mute,
		})
		jBytes, err := json.Marshal(taskObj)
		if err != nil {
			log.Println(err)
			return false
		}
		acked := make(chan bool)
		// now we wait for an ack with respect to actually performing the mute
		pubsub := tokenProvider.client.Subscribe(context.Background(), rediskey.CompleteTask(taskObj.TaskID))
		err = tokenProvider.client.Publish(context.Background(), rediskey.TasksSubscribe(connectCode), jBytes).Err()
		if err != nil {
			log.Println("Error in publishing task to " + rediskey.TasksSubscribe(connectCode))
			log.Println(err)
		} else {
			go tokenProvider.waitForAck(pubsub, timeout, acked)
			res := <-acked
			if res {
				log.Println("Successful mute/deafen using client capture bot!")

				// hooray! we did the mute with a client token!
				return true
			}
			err := tokenProvider.BlacklistTokenForDuration(guildID, connectCode, UnresponsiveCaptureBlacklistDuration)
			if err == nil {
				log.Printf("No ack from capture clients; blacklisting capture client for gamecode \"%s\" for %s\n", connectCode, UnresponsiveCaptureBlacklistDuration.String())
			}
		}
	} else {
		log.Println("Capture client is probably rate-limited. Deferring to main bot instead")
	}
	return false
}
