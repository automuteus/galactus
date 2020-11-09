package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

var ctx = context.Background()

type TokenProvider struct {
	client *redis.Client

	//maps hashed tokens to active discord sessions
	activeSessions map[string]*discordgo.Session
	sessionLock    sync.RWMutex
}

func guildTokensKey(guildID string) string {
	return "automuteus:tokens:" + guildID
}

func allTokensKey() string {
	return "automuteus:tokens"
}

func NewTokenProvider(redisAddr, redisUser, redisPass string, redisDB int) *TokenProvider {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPass,
		DB:       redisDB, // use default DB
	})
	return &TokenProvider{
		client:         rdb,
		activeSessions: make(map[string]*discordgo.Session),
		sessionLock:    sync.RWMutex{},
	}
}

func (tokenProvider *TokenProvider) PopulateAndStartSessions() {
	keys, err := tokenProvider.client.HGetAll(ctx, allTokensKey()).Result()
	if err != nil {
		log.Println(err)
		return
	}

	for _, v := range keys {
		k := hashToken(v)
		if _, ok := tokenProvider.activeSessions[k]; !ok {
			sess, err := discordgo.New("Bot " + v)
			if err != nil {
				log.Println(err)
				continue
			}
			err = sess.Open()
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println("Opened session on startup for " + k)
			tokenProvider.activeSessions[k] = sess
		}
	}
}

type NoNickPatchParams struct {
	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`
}

func (tokenProvider *TokenProvider) Run(port string) {
	r := mux.NewRouter()

	r.HandleFunc("/changestate/{guildID}/{userID}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		guildID := vars["guildID"]
		userID := vars["userID"]

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		pParams := NoNickPatchParams{}
		err = json.Unmarshal(body, &pParams)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		tokensKey := guildTokensKey(guildID)
		hashedToken, err := tokenProvider.client.SRandMember(ctx, tokensKey).Result()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if sess, ok := tokenProvider.activeSessions[hashedToken]; ok {
			log.Printf("Issuing update request to discord for UserID %s with mute=%v deaf=%v\n", userID, pParams.Mute, pParams.Deaf)

			_, err := sess.RequestWithBucketID("PATCH", discordgo.EndpointGuildMember(guildID, userID), pParams, discordgo.EndpointGuildMember(guildID, ""))
			if err != nil {
				log.Println(err)
				//guildMemberUpdateNoNick(s, params)
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

		token := string(body)

		k := hashToken(token)
		tokenProvider.sessionLock.RLock()
		if _, ok := tokenProvider.activeSessions[k]; ok {
			log.Println("Key already exists on the server")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Key already exists on the server"))
			tokenProvider.sessionLock.RUnlock()
			return
		}
		tokenProvider.sessionLock.RUnlock()

		sess, err := discordgo.New("Bot " + token)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		err = sess.Open()
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(err.Error()))
			return
		}

		hash := hashToken(token)
		tokenProvider.sessionLock.Lock()
		tokenProvider.activeSessions[hash] = sess
		tokenProvider.sessionLock.Unlock()

		sess.AddHandler(tokenProvider.newGuild())
		err = tokenProvider.client.HSet(ctx, allTokensKey(), hash, token).Err()
		if err != nil {
			log.Println(err)
		}

		//TODO handle guild removals?
		for _, v := range sess.State.Guilds {
			err := tokenProvider.client.SAdd(ctx, guildTokensKey(v.ID), hash).Err()
			if err != redis.Nil {
				log.Println(err)
			} else {
				log.Println("Added token for guild " + v.ID)
			}
		}
	}).Methods("POST")

	http.ListenAndServe(":"+port, r)
}

func hashToken(token string) string {
	h := sha256.New()
	h.Sum([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func (tokenProvider *TokenProvider) Close() {
	tokenProvider.sessionLock.Lock()
	for _, v := range tokenProvider.activeSessions {
		v.Close()
	}

	tokenProvider.activeSessions = map[string]*discordgo.Session{}
	tokenProvider.sessionLock.Unlock()
}

func (tokenProvider *TokenProvider) newGuild() func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {
		tokenProvider.sessionLock.RLock()
		for hashedToken, sess := range tokenProvider.activeSessions {
			if sess == s {
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
