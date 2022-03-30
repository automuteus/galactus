package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/automuteus/utils/pkg/premium"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/task"
	"github.com/automuteus/utils/pkg/token"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"golang.org/x/exp/constraints"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

type TokenProvider struct {
	client         *redis.Client
	primarySession *discordgo.Session

	// maps hashed tokens to active discord sessions
	activeSessions      map[string]*discordgo.Session
	maxRequests5Seconds int64
	sessionLock         sync.RWMutex

	botVerificationQueue chan botVerifyTask
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
		client:               rdb,
		primarySession:       dg,
		activeSessions:       make(map[string]*discordgo.Session),
		maxRequests5Seconds:  maxReq,
		sessionLock:          sync.RWMutex{},
		botVerificationQueue: make(chan botVerifyTask),
	}
}

func (tokenProvider *TokenProvider) BotVerificationWorker() {
	log.Println("Premium bot verification worker started")
	for {
		verifyTask := <-tokenProvider.botVerificationQueue

		if tokenProvider.canRunBotVerification(verifyTask.guildID) {
			// always send nil tokens used; we can't populate this info from anywhere anyways
			tokenProvider.verifyBotMembership(verifyTask.guildID, verifyTask.limit, nil)

			err := tokenProvider.markBotVerificationLockout(verifyTask.guildID)
			if err != nil {
				log.Println(err)
			}

			// cheap ratelimiting; only process verifications once per second
			time.Sleep(time.Second)
		}
	}
}

func rateLimitEventCallback(sess *discordgo.Session, rl *discordgo.RateLimit) {
	log.Println(rl.Message)
}

func (tokenProvider *TokenProvider) PopulateAndStartSessions(tokens []string) {
	for _, v := range tokens {
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
		sess.AddHandler(tokenProvider.newGuild)
		log.Println("Opened session on startup for " + k)
		tokenProvider.activeSessions[k] = sess
		return true
	}
	return false
}

func (tokenProvider *TokenProvider) getSession(guildID string, hTokenSubset map[string]struct{}) (*discordgo.Session, string) {
	tokenProvider.sessionLock.RLock()
	defer tokenProvider.sessionLock.RUnlock()

	for hToken, sess := range tokenProvider.activeSessions {
		// if we have already used this token successfully, or haven't set any restrictions
		if hTokenSubset == nil || mapHasEntry(hTokenSubset, hToken) {
			// if this token isn't potentially rate-limited
			if tokenProvider.IncrAndTestGuildTokenComboLock(guildID, hToken) {
				return sess, hToken
			} else {
				log.Println("Secondary token is potentially rate-limited. Skipping")
			}
		}
	}

	return nil, ""
}

func mapHasEntry[T constraints.Ordered, K any](dict map[T]K, key T) bool {
	if dict == nil {
		return false
	}
	_, ok := dict[key]
	return ok
}

func (tokenProvider *TokenProvider) IncrAndTestGuildTokenComboLock(guildID, hashToken string) bool {
	i, err := tokenProvider.client.Incr(context.Background(), rediskey.GuildTokenLock(guildID, hashToken)).Result()
	if err != nil {
		log.Println(err)
	}
	usable := i < tokenProvider.maxRequests5Seconds
	log.Printf("Token/capture %s on guild %s is at count %d. Using?: %v", hashToken, guildID, i, usable)
	if !usable {
		return false
	}

	err = tokenProvider.client.Expire(context.Background(), rediskey.GuildTokenLock(guildID, hashToken), time.Second*5).Err()
	if err != nil {
		log.Println(err)
	}

	return true
}

// BlacklistTokenForDuration sets a guild token (or connect code ala capture bot) to the maximum value allowed before
// attempting other non-rate-limited mute/deafen methods.
// NOTE: this will manifest as the capture/token in question appearing like it "has been used <maxnum> times" in logs,
// even if this is not technically accurate. A more accurate approach would probably use a totally separate Redis key,
// as opposed to this approach, which simply uses the ratelimiting counter key(s) to achieve blacklisting
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

		tasksChannel := make(chan task.UserModify, len(userModifications.Users))
		wg := sync.WaitGroup{}

		mdsc := task.MuteDeafenSuccessCounts{
			Worker:    0,
			Capture:   0,
			Official:  0,
			RateLimit: 0,
		}
		uniqueTokensUsed := make(map[string]struct{})
		lock := sync.Mutex{}
		tokenLock := sync.RWMutex{}

		errors := 0
		// start a handful of workers to handle the tasks
		for i := 0; i < maxWorkers; i++ {
			go func() {
				for request := range tasksChannel {
					userIDStr := strconv.FormatUint(request.UserID, 10)
					hToken := ""
					if limit > 0 {
						tokenLock.RLock()
						if len(uniqueTokensUsed) >= limit {
							hToken = tokenProvider.attemptOnSecondaryTokens(guildID, userIDStr, uniqueTokensUsed, request)
							tokenLock.RUnlock()
						} else {
							tokenLock.RUnlock()
							hToken = tokenProvider.attemptOnSecondaryTokens(guildID, userIDStr, nil, request)
						}
					}
					if hToken != "" {
						lock.Lock()
						mdsc.Worker++
						lock.Unlock()

						tokenLock.Lock()
						uniqueTokensUsed[hToken] = struct{}{}
						tokenLock.Unlock()
					} else {
						success := tokenProvider.attemptOnCaptureBot(guildID, connectCode, gid, taskTimeoutms, request)
						if success {
							lock.Lock()
							mdsc.Capture++
							lock.Unlock()
						} else {
							log.Printf("Applying mute=%v, deaf=%v using primary bot\n", request.Mute, request.Deaf)
							err = task.ApplyMuteDeaf(tokenProvider.primarySession, guildID, userIDStr, request.Mute, request.Deaf)
							if err != nil {
								lock.Lock()
								errors++
								lock.Unlock()
								log.Println("Error on primary bot:")
								log.Println(err)
							} else {
								lock.Lock()
								mdsc.Official++
								lock.Unlock()
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

		if errors > 0 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		jbytes, err := json.Marshal(mdsc)
		if err != nil {
			log.Println(err)
		} else {
			_, err := w.Write(jbytes)
			if err != nil {
				log.Println(err)
			}
		}

		// note, this should probably be more systematic on startup, not when a mute/deafen task comes in. But this is a
		// context in which we already have the guildID, successful tokens, AND the premium limit...
		go tokenProvider.verifyBotMembership(guildID, limit, uniqueTokensUsed)
	}).Methods("POST")

	r.HandleFunc("/verify/{guildID}/{premiumTier}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		guildID := vars["guildID"]
		tierStr := vars["premiumTier"]
		_, gerr := strconv.ParseUint(guildID, 10, 64)
		if gerr != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid guildID (non-numeric) received. Query should be of the form POST `/verify/<guildID>/<premiumTier>`"))
			return
		}
		tier, perr := strconv.ParseUint(tierStr, 10, 64)
		if perr != nil || tier < 0 || tier > 5 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid premium tier (not [0,5]) received. Query should be of the form POST `/verify/<guildID>/<premiumTier>`"))
			return
		}
		limit := PremiumBotConstraints[premium.Tier(tier)]
		tokenProvider.enqueueBotMembershipVerifyTask(guildID, limit)
		w.WriteHeader(http.StatusOK)
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

func (tokenProvider *TokenProvider) newGuild(s *discordgo.Session, m *discordgo.GuildCreate) {
	log.Println("added to " + m.ID)
}
