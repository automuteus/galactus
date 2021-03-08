package galactus

import (
	"context"
	"github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/utils/pkg/discord"
	"github.com/automuteus/utils/pkg/rediskey"
	"go.uber.org/zap"
	"time"
)

func (galactus *GalactusAPI) attemptOnSecondaryTokens(guildID, userID string, tokens []string, limit int, request discord.UserModify) bool {
	if tokens != nil && limit > 0 {
		sess, hToken := galactus.getAnySession(guildID, tokens, limit)
		if sess != nil {
			err := discord.ApplyMuteDeaf(sess, guildID, userID, request.Mute, request.Deaf)
			if err != nil {
				galactus.logger.Error("failed to apply mute/deaf on secondary bot",
					zap.Error(err),
					zap.String("guildID", guildID),
					zap.Uint64("userID", request.UserID),
					zap.String("hashedToken", hToken),
					zap.Bool("mute", request.Mute),
					zap.Bool("deaf", request.Deaf),
				)
			} else {
				galactus.logger.Info("successfully applied mute/deaf on secondary bot",
					zap.Error(err),
					zap.String("guildID", guildID),
					zap.Uint64("userID", request.UserID),
					zap.String("hashedToken", hToken),
					zap.Bool("mute", request.Mute),
					zap.Bool("deaf", request.Deaf),
				)
				return true
			}
		} else {
			galactus.logger.Info("no secondary bot tokens found",
				zap.String("guildID", guildID),
				zap.String("userID", userID),
			)
		}
	} else {
		galactus.logger.Info("guild has no access to secondary bot tokens; skipping",
			zap.String("guildID", guildID),
		)
	}
	return false
}

var UnresponsiveCaptureBlacklistDuration = time.Minute * time.Duration(5)

func (galactus *GalactusAPI) attemptOnCaptureBot(guildID, connectCode string, gid uint64, timeout time.Duration, request discord.UserModify) bool {
	// this is cheeky, but use the connect code as part of the lock; don't issue too many requests on the capture client w/ this code
	if galactus.IncrAndTestGuildTokenComboLock(guildID, connectCode) {
		// if the secondary token didn't work, then next we try the client-side capture request
		taskObj := discord.NewModifyTask(gid, request.UserID, discord.PatchParams{
			Deaf: request.Deaf,
			Mute: request.Mute,
		})

		acked := make(chan bool)
		// now we wait for an ack with respect to actually performing the mute
		pubsub := galactus.client.Subscribe(context.Background(), rediskey.CompleteTask(taskObj.TaskID))
		defer pubsub.Close()

		err := redis.PushCaptureClientTask(galactus.client, connectCode, taskObj, timeout)
		if err != nil {
			galactus.logger.Error("error pushing capture client task",
				zap.Error(err),
				zap.String("key", rediskey.TasksList(connectCode)))
		} else {
			go galactus.waitForAck(pubsub, timeout, acked)
			res := <-acked
			if res {
				galactus.logger.Info("successful mute/deafen using client capture bot",
					zap.String("taskID", taskObj.TaskID),
				)
				// hooray! we did the mute with a client token!
				return true
			}
			err := galactus.BlacklistTokenForDuration(guildID, connectCode, UnresponsiveCaptureBlacklistDuration)
			if err == nil {
				galactus.logger.Info("no ack from capture clients. Not using capture client for a time period",
					zap.String("connectCode", connectCode),
					zap.String("duration", UnresponsiveCaptureBlacklistDuration.String()),
				)
			}
		}
	} else {
		galactus.logger.Info("capture client likely rate-limited or refusing tasks. Using main bot instead")
	}
	return false
}
