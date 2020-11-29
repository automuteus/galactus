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

const BroadcastToClientCapturesTimeout = time.Millisecond * 500
const AckFromClientCapturesTimeout = time.Second

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
	return "automuteus:lock:" + hToken + ":" + guildID
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

func (tokenProvider *TokenProvider) getAnySession(guildID string, limit int) (*discordgo.Session, string) {
	hTokens, err := tokenProvider.client.SMembers(context.Background(), guildTokensKey(guildID)).Result()
	if err != nil {
		return nil, ""
	}

	tokenProvider.sessionLock.RLock()
	defer tokenProvider.sessionLock.RUnlock()

	for i, hToken := range hTokens {
		if i == limit {
			return nil, ""
		}
		//if this token isn't potentially rate-limited
		if tokenProvider.CanUseGuildTokenCombo(guildID, hToken) {
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

func (tokenProvider *TokenProvider) IncrGuildTokenComboLock(guildID, hashToken string) {
	err := tokenProvider.client.Incr(context.Background(), guildTokenLock(guildID, hashToken)).Err()
	if err != nil {
		log.Println()
	}
	tokenProvider.client.Expire(context.Background(), guildTokenLock(guildID, hashToken), time.Second*5)
}

func (tokenProvider *TokenProvider) CanUseGuildTokenCombo(guildID, hashToken string) bool {
	res, err := tokenProvider.client.Get(context.Background(), guildTokenLock(guildID, hashToken)).Result()
	if err == redis.Nil {
		return true
	} else if err != nil {
		log.Println(err)
		return true
	}
	i, err := strconv.ParseInt(res, 10, 64)
	if err != nil {
		log.Println(err)
		return true
	}

	return i < tokenProvider.maxRequests5Seconds
}

func (tokenProvider *TokenProvider) BlacklistTokenForDuration(guildID, hashToken string, duration time.Duration) error {
	return tokenProvider.client.Set(context.Background(), guildTokenLock(guildID, hashToken), tokenProvider.maxRequests5Seconds, duration).Err()
}

var UnresponsiveCaptureBlacklistDuration = time.Minute * time.Duration(1)

func (tokenProvider *TokenProvider) Run(port string) {
	r := mux.NewRouter()

	// /modify/guild/conncode/userid?mute=true?deaf=false
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

		wg := sync.WaitGroup{}
		mdsc := discord.MuteDeafenSuccessCounts{
			Worker:   0,
			Capture:  0,
			Official: 0,
		}
		mdscLock := sync.Mutex{}

		for _, modifyReq := range userModifications.Users {
			wg.Add(1)

			go func(request UserModify) {
				defer wg.Done()

				userIdStr := strconv.FormatUint(request.UserID, 10)

				limit := PremiumBotConstraints[userModifications.Premium]
				if limit > 0 {
					sess, hToken := tokenProvider.getAnySession(guildID, limit)
					if sess != nil {
						err := discord.ApplyMuteDeaf(sess, guildID, userIdStr, request.Mute, request.Deaf)
						if err == nil {
							tokenProvider.IncrGuildTokenComboLock(guildID, hToken)
							log.Printf("Successfully applied mute=%v, deaf=%v to User %d using secondary bot: %s\n", request.Mute, request.Deaf, request.UserID, hToken)
							mdscLock.Lock()
							mdsc.Worker++
							mdscLock.Unlock()
							return
						}
					} else {
						log.Println("No secondary bot tokens found. Trying other methods")
					}
				} else {
					log.Println("Guild has no access to secondary bot tokens; skipping")
				}
				//this is cheeky, but use the connect code as part of the lock; don't issue too many requests on the capture client w/ this code
				if tokenProvider.CanUseGuildTokenCombo(guildID, connectCode) {
					//if the secondary token didn't work, then next we try the client-side capture request
					task := discord.NewModifyTask(gid, request.UserID, discord.NoNickPatchParams{
						Deaf: request.Deaf,
						Mute: request.Mute,
					})
					jBytes, err := json.Marshal(task)
					if err != nil {
						log.Println(err)
						return
					} else {
						acked := make(chan bool)
						pubsub := tokenProvider.client.Subscribe(context.Background(), discord.BroadcastTaskAckKey(task.TaskID))
						go tokenProvider.waitForAck(pubsub, BroadcastToClientCapturesTimeout, acked)

						err := tokenProvider.client.Publish(context.Background(), discord.TasksSubscribeKey(connectCode), jBytes).Err()
						if err != nil {
							log.Println(err)
						}

						res := <-acked
						if !res {
							err := tokenProvider.BlacklistTokenForDuration(guildID, connectCode, UnresponsiveCaptureBlacklistDuration)
							if err == nil {
								log.Printf("No ack from capture clients; blacklisting capture client for gamecode \"%s\" for %s\n", connectCode, UnresponsiveCaptureBlacklistDuration.String())
							}
							//falls through to using official bot token below
						} else {
							acked := make(chan bool)
							pubsub := tokenProvider.client.Subscribe(context.Background(), discord.CompleteTaskAckKey(task.TaskID))
							go tokenProvider.waitForAck(pubsub, AckFromClientCapturesTimeout, acked)
							res := <-acked
							if res {
								log.Println("Successful mute/deafen using client capture bot!")
								tokenProvider.IncrGuildTokenComboLock(guildID, connectCode)
								mdscLock.Lock()
								mdsc.Capture++
								mdscLock.Unlock()
								//hooray! we did the mute with a client token!
								return
							} else {
								err := tokenProvider.BlacklistTokenForDuration(guildID, connectCode, UnresponsiveCaptureBlacklistDuration)
								if err == nil {
									log.Printf("No ack from capture clients; blacklisting capture client for gamecode \"%s\" for %s\n", connectCode, UnresponsiveCaptureBlacklistDuration.String())
								}
								//falls through to using official bot token below
							}
						}
					}
				} else {
					log.Println("Capture client is probably rate-limited. Deferring to main bot instead")
				}
				log.Printf("Applying mute=%v, deaf=%v using primary bot\n", request.Mute, request.Deaf)
				err = discord.ApplyMuteDeaf(tokenProvider.primarySession, guildID, userIdStr, request.Mute, request.Deaf)
				if err != nil {
					log.Println(err)
				}
				mdscLock.Lock()
				mdsc.Official++
				mdscLock.Unlock()

			}(modifyReq)
		}
		wg.Wait()
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

func (tokenProvider *TokenProvider) waitForAck(pubsub *redis.PubSub, waitTime time.Duration, result chan<- bool) {
	t := time.NewTimer(waitTime)
	defer pubsub.Close()

	for {
		select {
		case <-t.C:
			result <- false
			return
		case t := <-pubsub.Channel():
			result <- t.Payload == "true"
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
