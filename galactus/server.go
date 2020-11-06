package galactus

import (
	"context"
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

	//maps tokens to active discord sessions
	activeSessions map[string]*discordgo.Session
	sessionLock    sync.RWMutex
}

func guildTokensKey(guildID string) string {
	return "automuteus:tokens:" + guildID
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

type NoNickPatchParams struct {
	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`
}

func (tokenProvider *TokenProvider) Run(port string) {
	r := mux.NewRouter()

	r.HandleFunc("/v1/changestate/{guildID}/{userID}", func(w http.ResponseWriter, r *http.Request) {
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
		botToken, err := tokenProvider.client.SRandMember(ctx, tokensKey).Result()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if sess, ok := tokenProvider.activeSessions[botToken]; ok {
			log.Printf("Issuing update request to discord for UserID %s with mute=%v deaf=%v\n", userID, pParams.Mute, pParams.Deaf)

			_, err := sess.RequestWithBucketID("PATCH", discordgo.EndpointGuildMember(guildID, userID), pParams, discordgo.EndpointGuildMember(guildID, ""))
			if err != nil {
				log.Println("Failed to change nickname for User: move the bot up in your Roles")
				log.Println(err)
				//guildMemberUpdateNoNick(s, params)
			}
		}
	}).Methods("POST")

	r.HandleFunc("/v1/addtoken", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()

		token := string(body)
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

		tokenProvider.sessionLock.Lock()
		tokenProvider.activeSessions[token] = sess
		tokenProvider.sessionLock.Unlock()

		//TODO need to also guarantee that additions while already running are handled, as well as guild removals
		for _, v := range sess.State.Guilds {
			err := tokenProvider.client.SAdd(ctx, guildTokensKey(v.ID), token).Err()
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Added token for guild " + v.ID)
			}
		}
	}).Methods("POST")

	http.ListenAndServe(":"+port, r)
}

func (tokenProvider *TokenProvider) Close() {
	tokenProvider.sessionLock.Lock()
	for _, v := range tokenProvider.activeSessions {
		v.Close()
	}
	tokenProvider.activeSessions = map[string]*discordgo.Session{}
	tokenProvider.sessionLock.Unlock()
}
