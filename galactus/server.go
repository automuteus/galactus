package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/automuteus/galactus/discord"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultCaptureBotTimeout = time.Second

var DefaultIdentifyThresholds = discord.IdentifyThresholds{
	HardWindow:    time.Hour * 24,
	HardThreshold: 950,
	SoftWindow:    time.Hour * 12,
	SoftThreshold: 500,
}

var ctx = context.Background()

type TokenProvider struct {
	client         *redis.Client
	primarySession *discordgo.Session

	//maps hashed tokens to active discord sessions
	activeSessions      map[string]*discordgo.Session
	maxRequests5Seconds int64
	sessionLock         sync.RWMutex
}

func guildTokensKey(guildID string) string {
	return "automuteus:tokens:guild:" + guildID
}

func allTokensKey() string {
	return "automuteus:alltokens"
}

func guildTokenLock(guildID, hToken string) string {
	return "automuteus:muterequest:lock:" + hToken + ":" + guildID
}

func NewTokenProvider(botToken, redisAddr, redisUser, redisPass string, maxReq int64) *TokenProvider {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPass,
		DB:       0, // use default DB
	})

	if discord.IsTokenLockedOut(rdb, botToken, DefaultIdentifyThresholds) {
		log.Fatal("BOT HAS EXCEEDED TOKEN LOCKOUT ON PRIMARY TOKEN")
	}

	discord.WaitForToken(rdb, botToken)
	discord.MarkIdentifyAndLockForToken(rdb, botToken)

	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds)
	shards := os.Getenv("NUM_SHARDS")
	if shards != "" {
		n, err := strconv.ParseInt(shards, 10, 64)
		if err != nil {
			log.Println(err)
		}
		dg.ShardCount = int(n)
		dg.ShardID = 0
	}
	dg.AddHandler(rateLimitEventCallback)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	return &TokenProvider{
		client:              rdb,
		primarySession:      dg,
		activeSessions:      make(map[string]*discordgo.Session),
		maxRequests5Seconds: maxReq,
		sessionLock:         sync.RWMutex{},
	}
}

func rateLimitEventCallback(sess *discordgo.Session, rl *discordgo.RateLimit) {
	log.Println(rl.Message)
}

func (tokenProvider *TokenProvider) PopulateAndStartSessions() {
	keys, err := tokenProvider.client.HGetAll(ctx, allTokensKey()).Result()
	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range keys {
		tokenProvider.openAndStartSessionWithToken(v)
	}
}

func (tokenProvider *TokenProvider) openAndStartSessionWithToken(token string) bool {
	k := hashToken(token)
	tokenProvider.sessionLock.Lock()
	defer tokenProvider.sessionLock.Unlock()

	if _, ok := tokenProvider.activeSessions[k]; !ok {
		if discord.IsTokenLockedOut(tokenProvider.client, token, DefaultIdentifyThresholds) {
			log.Println("Token <redacted> is locked out!")
			return false
		}
		discord.WaitForToken(tokenProvider.client, token)
		discord.MarkIdentifyAndLockForToken(tokenProvider.client, token)
		sess, err := discordgo.New("Bot " + token)
		if err != nil {
			log.Println(err)
			return false
		}
		sess.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds)
		err = sess.Open()
		if err != nil {
			log.Println(err)
			return false
		}
		//associates the guilds with this token to be used for requests
		sess.AddHandler(tokenProvider.newGuild(k))
		log.Println("Opened session on startup for " + k)
		tokenProvider.activeSessions[k] = sess
		return true
	}
	return false
}

func (tokenProvider *TokenProvider) getAllTokensForGuild(guildID string) []string {
	hTokens, err := tokenProvider.client.SMembers(context.Background(), guildTokensKey(guildID)).Result()
	if err != nil {
		return nil
	}
	return hTokens
}

func (tokenProvider *TokenProvider) getAnySession(guildID string, tokens []string, limit int) (*discordgo.Session, string) {
	tokenProvider.sessionLock.RLock()
	defer tokenProvider.sessionLock.RUnlock()

	for i, hToken := range tokens {
		if i == limit {
			return nil, ""
		}
		//if this token isn't potentially rate-limited
		if tokenProvider.IncrAndTestGuildTokenComboLock(guildID, hToken) {
			if sess, ok := tokenProvider.activeSessions[hToken]; ok {
				return sess, hToken
			} else {
				//remove this key from our records and keep going
				tokenProvider.client.SRem(context.Background(), guildTokensKey(guildID), hToken)
			}
		} else {
			log.Println("Secondary token is potentially rate-limited. Skipping")
		}
	}

	return nil, ""
}

func (tokenProvider *TokenProvider) IncrAndTestGuildTokenComboLock(guildID, hashToken string) bool {
	i, err := tokenProvider.client.Incr(context.Background(), guildTokenLock(guildID, hashToken)).Result()
	if err != nil {
		log.Println(err)
	}
	usable := i < tokenProvider.maxRequests5Seconds
	log.Printf("Token %s on guild %s is at count %d. Using: %v", hashToken, guildID, i, usable)
	if !usable {
		return false
	}

	err = tokenProvider.client.Expire(context.Background(), guildTokenLock(guildID, hashToken), time.Second*5).Err()
	if err != nil {
		log.Println(err)
	}

	return true
}

func (tokenProvider *TokenProvider) BlacklistTokenForDuration(guildID, hashToken string, duration time.Duration) error {
	return tokenProvider.client.Set(context.Background(), guildTokenLock(guildID, hashToken), tokenProvider.maxRequests5Seconds, duration).Err()
}

const DefaultMaxWorkers = 8

var UnresponsiveCaptureBlacklistDuration = time.Minute * time.Duration(5)

func (tokenProvider *TokenProvider) Run(port string) {
	r := mux.NewRouter()

	taskTimeoutms := DefaultCaptureBotTimeout

	taskTimeoutmsStr := os.Getenv("ACK_TIMEOUT_MS")
	num, err := strconv.ParseInt(taskTimeoutmsStr, 10, 64)
	if err == nil {
		log.Printf("Read from env; using ACK_TIMEOUT_MS=%d\n", num)
		taskTimeoutms = time.Millisecond * time.Duration(num)
	}

	maxWorkers := DefaultMaxWorkers
	maxWorkersStr := os.Getenv("MAX_WORKERS")
	num, err = strconv.ParseInt(maxWorkersStr, 10, 64)
	if err == nil {
		log.Printf("Read from env; using MAX_WORKERS=%d\n", num)
		maxWorkers = int(num)
	}

	r.HandleFunc("/modify/{guildID}/{connectCode}", func(w http.ResponseWriter, r *http.Request) {
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

		userModifications := UserModifyRequest{}
		err = json.Unmarshal(body, &userModifications)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		limit := PremiumBotConstraints[userModifications.Premium]
		tokens := tokenProvider.getAllTokensForGuild(guildID)

		tasksChannel := make(chan UserModify, len(userModifications.Users))
		wg := sync.WaitGroup{}

		mdsc := discord.MuteDeafenSuccessCounts{
			Worker:    0,
			Capture:   0,
			Official:  0,
			RateLimit: 0,
		}
		mdscLock := sync.Mutex{}

		//start a handful of workers to handle the tasks
		for i := 0; i < maxWorkers; i++ {
			go func() {
				for request := range tasksChannel {
					userIdStr := strconv.FormatUint(request.UserID, 10)
					success := tokenProvider.attemptOnSecondaryTokens(guildID, userIdStr, tokens, limit, request)
					if success {
						mdscLock.Lock()
						mdsc.Worker++
						mdscLock.Unlock()
					} else {
						success = tokenProvider.attemptOnCaptureBot(guildID, connectCode, gid, taskTimeoutms, request)
						if success {
							mdscLock.Lock()
							mdsc.Capture++
							mdscLock.Unlock()
						} else {
							log.Printf("Applying mute=%v, deaf=%v using primary bot\n", request.Mute, request.Deaf)
							err = discord.ApplyMuteDeaf(tokenProvider.primarySession, guildID, userIdStr, request.Mute, request.Deaf)
							if err != nil {
								log.Println(err)
							} else {
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
		return
	}).Methods("POST")

	r.HandleFunc("/addtoken", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		token := string(body)
		log.Println(token)

		k := hashToken(token)
		log.Println(k)
		tokenProvider.sessionLock.RLock()
		if _, ok := tokenProvider.activeSessions[k]; ok {
			log.Println("Token already exists on the server")
			w.WriteHeader(http.StatusAlreadyReported)
			w.Write([]byte("Token already exists on the server"))
			tokenProvider.sessionLock.RUnlock()
			return
		}
		tokenProvider.sessionLock.RUnlock()

		if discord.IsTokenLockedOut(tokenProvider.client, token, DefaultIdentifyThresholds) {
			log.Println("Token <redacted> is locked out!")
			return
		}
		discord.WaitForToken(tokenProvider.client, token)
		discord.MarkIdentifyAndLockForToken(tokenProvider.client, token)
		sess, err := discordgo.New("Bot " + token)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		sess.AddHandler(tokenProvider.newGuild(k))
		err = sess.Open()
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}

		tokenProvider.sessionLock.Lock()
		tokenProvider.activeSessions[k] = sess
		tokenProvider.sessionLock.Unlock()

		err = tokenProvider.client.HSet(ctx, allTokensKey(), k, token).Err()
		if err != nil {
			log.Println(err)
		}

		for _, v := range sess.State.Guilds {
			err := tokenProvider.client.SAdd(ctx, guildTokensKey(v.ID), k).Err()
			if err != redis.Nil && err != nil {
				log.Println(strings.ReplaceAll(err.Error(), token, "<redacted>"))
			} else {
				log.Println("Added token for guild " + v.ID)
			}
		}
	}).Methods("POST")

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}).Methods("GET")

	log.Println("Galactus token service is running on port " + port + "...")
	http.ListenAndServe(":"+port, r)
}

func (tokenProvider *TokenProvider) attemptOnSecondaryTokens(guildID, userID string, tokens []string, limit int, request UserModify) bool {
	if tokens != nil && limit > 0 {
		sess, hToken := tokenProvider.getAnySession(guildID, tokens, limit)
		if sess != nil {
			err := discord.ApplyMuteDeaf(sess, guildID, userID, request.Mute, request.Deaf)
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

func (tokenProvider *TokenProvider) attemptOnCaptureBot(guildID, connectCode string, gid uint64, timeout time.Duration, request UserModify) bool {
	//this is cheeky, but use the connect code as part of the lock; don't issue too many requests on the capture client w/ this code
	if tokenProvider.IncrAndTestGuildTokenComboLock(guildID, connectCode) {
		//if the secondary token didn't work, then next we try the client-side capture request
		task := discord.NewModifyTask(gid, request.UserID, discord.NoNickPatchParams{
			Deaf: request.Deaf,
			Mute: request.Mute,
		})
		jBytes, err := json.Marshal(task)
		if err != nil {
			log.Println(err)
			return false
		} else {
			acked := make(chan bool)
			//now we wait for an ack with respect to actually performing the mute
			pubsub := tokenProvider.client.Subscribe(context.Background(), discord.CompleteTaskAckKey(task.TaskID))

			err := tokenProvider.client.Publish(context.Background(), discord.TasksSubscribeKey(connectCode), jBytes).Err()
			if err != nil {
				log.Println("Error in publishing task to " + discord.TasksSubscribeKey(connectCode))
				log.Println(err)
			} else {
				go tokenProvider.waitForAck(pubsub, timeout, acked)
				res := <-acked
				if res {
					log.Println("Successful mute/deafen using client capture bot!")

					//hooray! we did the mute with a client token!
					return true
				} else {
					err := tokenProvider.BlacklistTokenForDuration(guildID, connectCode, UnresponsiveCaptureBlacklistDuration)
					if err == nil {
						log.Printf("No ack from capture clients; blacklisting capture client for gamecode \"%s\" for %s\n", connectCode, UnresponsiveCaptureBlacklistDuration.String())
					}
				}
			}
		}
	} else {
		log.Println("Capture client is probably rate-limited. Deferring to main bot instead")
	}
	return false
}

func (tokenProvider *TokenProvider) rateLimitEventCallback(sess *discordgo.Session, rl *discordgo.RateLimit) {
	log.Println(rl.Message)
}

func (tokenProvider *TokenProvider) waitForAck(pubsub *redis.PubSub, waitTime time.Duration, result chan<- bool) {
	t := time.NewTimer(waitTime)
	defer pubsub.Close()
	channel := pubsub.Channel()

	for {
		select {
		case <-t.C:
			t.Stop()
			result <- false
			return
		case val := <-channel:
			t.Stop()
			result <- val.Payload == "true"
			return
		}
	}
}

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (tokenProvider *TokenProvider) Close() {
	tokenProvider.sessionLock.Lock()
	for _, v := range tokenProvider.activeSessions {
		v.Close()
	}

	tokenProvider.activeSessions = map[string]*discordgo.Session{}
	tokenProvider.sessionLock.Unlock()
	tokenProvider.primarySession.Close()
}

func (tokenProvider *TokenProvider) newGuild(hashedToken string) func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {
		tokenProvider.sessionLock.RLock()
		for test := range tokenProvider.activeSessions {
			if hashedToken == test {
				err := tokenProvider.client.SAdd(ctx, guildTokensKey(m.Guild.ID), hashedToken)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("Token added for running guild " + m.Guild.ID)
				}
			}
		}

		tokenProvider.sessionLock.RUnlock()
	}
}
