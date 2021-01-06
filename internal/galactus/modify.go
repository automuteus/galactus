package galactus

import (
	"encoding/json"
	"github.com/automuteus/utils/pkg/task"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

func (galactus *GalactusAPI) modifyUserHandler(maxWorkers int, taskTimeout time.Duration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		guildID := vars["guildID"]
		connectCode := vars["connectCode"]
		gid, gerr := strconv.ParseUint(guildID, 10, 64)
		if gerr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid guildID received. Query should be of the form POST `/modify/<guildID>/<conncode>`"))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		userModifications := task.UserModifyRequest{}
		err = json.Unmarshal(body, &userModifications)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		limit := PremiumBotConstraints[userModifications.Premium]
		tokens := galactus.getAllTokensForGuild(guildID)

		tasksChannel := make(chan task.UserModify, len(userModifications.Users))
		wg := sync.WaitGroup{}

		mdsc := task.MuteDeafenSuccessCounts{
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
							sess, err := getRandomSession(galactus.shardManager)
							if err != nil {
								galactus.logger.Error("error fetching random session for user modify",
									zap.Error(err),
									zap.String("guildID", guildID),
									zap.String("userID", userIDStr),
									zap.Bool("mute", request.Mute),
									zap.Bool("deaf", request.Deaf),
								)
							}

							err = task.ApplyMuteDeaf(sess, guildID, userIDStr, request.Mute, request.Deaf)
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
			log.Println(err)
		} else {
			_, err := w.Write(jbytes)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
