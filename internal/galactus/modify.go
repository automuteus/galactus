package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/discord"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func (galactus *GalactusAPI) modifyUserHandler(maxWorkers int, taskTimeout time.Duration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.ModifyUserFull)
		connectCode := validate.ConnectCodeAndRespond(galactus.logger, w, r, endpoint.ModifyUserFull)

		if guildID == "" || connectCode == "" {
			return
		}

		// We can safely ignore the error here, because we already validated the snowflake above
		gid, _ := strconv.ParseUint(guildID, 10, 64)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			galactus.logger.Error("failed to read HTTP request body",
				zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		userModifications := discord.UserModifyRequest{}
		err = json.Unmarshal(body, &userModifications)
		if err != nil {
			galactus.logger.Error("failed to unmarshal user modification request",
				zap.Error(err))
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		limit := PremiumBotConstraints[userModifications.Premium]
		tokens := galactus.getAllTokensForGuild(guildID)

		tasksChannel := make(chan discord.UserModify, len(userModifications.Users))
		wg := sync.WaitGroup{}

		mdsc := discord.MuteDeafenSuccessCounts{
			Worker:    0,
			Capture:   0,
			Official:  0,
			RateLimit: 0,
		}
		mdscLock := sync.Mutex{}

		// start a handful of workers to handle the tasks
		for i := 0; i < maxWorkers; i++ {
			go func() {
				for request := range tasksChannel {
					userIDStr := strconv.FormatUint(request.UserID, 10)
					success := galactus.attemptOnSecondaryTokens(guildID, userIDStr, tokens, limit, request)
					if success {
						mdscLock.Lock()
						mdsc.Worker++
						mdscLock.Unlock()
					} else {
						success = galactus.attemptOnCaptureBot(guildID, connectCode, gid, taskTimeout, request)
						if success {
							mdscLock.Lock()
							mdsc.Capture++
							mdscLock.Unlock()
						} else {
							sess := galactus.shardManager.Session(0)
							if sess == nil {
								galactus.logger.Error("error fetching session 0 for user modify",
									zap.String("guildID", guildID),
									zap.String("userID", userIDStr),
									zap.Bool("mute", request.Mute),
									zap.Bool("deaf", request.Deaf),
								)
							}

							err = discord.ApplyMuteDeaf(sess, guildID, userIDStr, request.Mute, request.Deaf)
							if err != nil {
								galactus.logger.Error("error applying mute/deaf on official bot",
									zap.Error(err),
									zap.String("guildID", guildID),
									zap.String("userID", userIDStr),
									zap.Bool("mute", request.Mute),
									zap.Bool("deaf", request.Deaf),
								)
							} else {
								galactus.logger.Error("successfully applied mute/deaf on official bot",
									zap.String("guildID", guildID),
									zap.String("userID", userIDStr),
									zap.Bool("mute", request.Mute),
									zap.Bool("deaf", request.Deaf),
								)
								mdscLock.Lock()
								mdsc.Official++
								mdscLock.Unlock()
							}
						}
					}
					wg.Done()
				}
			}()
		}

		for _, modifyReq := range userModifications.Users {
			wg.Add(1)
			tasksChannel <- modifyReq
		}
		wg.Wait()
		close(tasksChannel)

		w.WriteHeader(http.StatusOK)
		jbytes, err := json.Marshal(mdsc)
		if err != nil {
			galactus.logger.Error("failed to marshal mutedeafensuccesscounts to JSON",
				zap.Error(err))
		} else {
			_, err := w.Write(jbytes)
			if err != nil {
				galactus.logger.Error("failed to write out json response",
					zap.Error(err))
			}
		}
		RecordDiscordRequests(galactus.client, MuteDeafenOfficial, mdsc.Official)
		RecordDiscordRequests(galactus.client, MuteDeafenWorker, mdsc.Worker)
		RecordDiscordRequests(galactus.client, MuteDeafenCapture, mdsc.Capture)
	}
}
