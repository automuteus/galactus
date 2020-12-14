package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/automuteus/utils/pkg/premium"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/task"
	"github.com/automuteus/utils/pkg/token"
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

var PremiumBotConstraints = map[premium.Tier]int{
	0: 0,
	1: 0,   // Free and Bronze have no premium bots
	2: 1,   // Silver has 1 bot
	3: 3,   // Gold has 3 bots
	4: 10,  // Platinum (TBD)
	5: 100, // Selfhost; 100 bots(!)
}

const DefaultCaptureBotTimeout = time.Second

var ctx = context.Background()

type TokenProvider struct {
	client         *redis.Client
	primarySession *discordgo.Session

	// maps hashed tokens to active discord sessions
	activeSessions      map[string]*discordgo.Session
	maxRequests5Seconds int64
	sessionLock         sync.RWMutex
}

func NewTokenProvider(botToken, redisAddr, redisUser, redisPass string, maxReq int64) *TokenProvider {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPass,
		DB:       0, // use default DB
	})

	token.WaitForToken(rdb, botToken)
	token.LockForToken(rdb, botToken)

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
	keys, err := tokenProvider.client.HGetAll(ctx, rediskey.AllTokensHSet).Result()
	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range keys {
		tokenProvider.openAndStartSessionWithToken(v)
	}
}

func (tokenProvider *TokenProvider) openAndStartSessionWithToken(botToken string) bool {
	k := hashToken(botToken)
	tokenProvider.sessionLock.Lock()
	defer tokenProvider.sessionLock.Unlock()

	if _, ok := tokenProvider.activeSessions[k]; !ok {
		token.WaitForToken(tokenProvider.client, botToken)
		token.LockForToken(tokenProvider.client, botToken)
		sess, err := discordgo.New("Bot " + botToken)
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
		// associates the guilds with this token to be used for requests
		sess.AddHandler(tokenProvider.newGuild(k))
		log.Println("Opened session on startup for " + k)
		tokenProvider.activeSessions[k] = sess
		return true
	}
	return false
}

func (tokenProvider *TokenProvider) getAllTokensForGuild(guildID string) []string {
	hTokens, err := tokenProvider.client.SMembers(context.Background(), rediskey.GuildTokensKey(guildID)).Result()
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
		// if this token isn't potentially rate-limited
		if tokenProvider.IncrAndTestGuildTokenComboLock(guildID, hToken) {
			sess, ok := tokenProvider.activeSessions[hToken]
			if ok {
				return sess, hToken
			}
			// remove this key from our records and keep going
			tokenProvider.client.SRem(context.Background(), rediskey.GuildTokensKey(guildID), hToken)
		} else {
			log.Println("Secondary token is potentially rate-limited. Skipping")
		}
	}

	return nil, ""
}

func (tokenProvider *TokenProvider) IncrAndTestGuildTokenComboLock(guildID, hashToken string) bool {
	i, err := tokenProvider.client.Incr(context.Background(), rediskey.GuildTokenLock(guildID, hashToken)).Result()
	if err != nil {
		log.Println(err)
	}
	usable := i < tokenProvider.maxRequests5Seconds
	log.Printf("Token %s on guild %s is at count %d. Using: %v", hashToken, guildID, i, usable)
	if !usable {
		return false
	}

	err = tokenProvider.client.Expire(context.Background(), rediskey.GuildTokenLock(guildID, hashToken), time.Second*5).Err()
	if err != nil {
		log.Println(err)
	}

	return true
}

func (tokenProvider *TokenProvider) BlacklistTokenForDuration(guildID, hashToken string, duration time.Duration) error {
	return tokenProvider.client.Set(context.Background(), rediskey.GuildTokenLock(guildID, hashToken), tokenProvider.maxRequests5Seconds, duration).Err()
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

		userModifications := task.UserModifyRequest{}
		err = json.Unmarshal(body, &userModifications)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		limit := PremiumBotConstraints[userModifications.Premium]
		tokens := tokenProvider.getAllTokensForGuild(guildID)

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
					success := tokenProvider.attemptOnSecondaryTokens(guildID, userIDStr, tokens, limit, request)
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
							err = task.ApplyMuteDeaf(tokenProvider.primarySession, guildID, userIDStr, request.Mute, request.Deaf)
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

		botToken := string(body)
		log.Println(botToken)

		k := hashToken(botToken)
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

		token.WaitForToken(tokenProvider.client, botToken)
		token.LockForToken(tokenProvider.client, botToken)
		sess, err := discordgo.New("Bot " + botToken)
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

		err = tokenProvider.client.HSet(ctx, rediskey.AllTokensHSet, k, botToken).Err()
		if err != nil {
			log.Println(err)
		}

		for _, v := range sess.State.Guilds {
			err := tokenProvider.client.SAdd(ctx, rediskey.GuildTokensKey(v.ID), k).Err()
			if !errors.Is(err, redis.Nil) && err != nil {
				log.Println(strings.ReplaceAll(err.Error(), botToken, "<redacted>"))
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
				err := tokenProvider.client.SAdd(ctx, rediskey.GuildTokensKey(m.Guild.ID), hashedToken)
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
