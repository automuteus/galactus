package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	redisutils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/premium"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/storage"
	"github.com/automuteus/utils/pkg/token"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/jonas747/dshardmanager"
	"github.com/top-gg/go-dbl"
	"go.uber.org/zap"
	"net/http"
	"os"
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

var DefaultIntents = discordgo.MakeIntent(discordgo.IntentsGuildVoiceStates | discordgo.IntentsGuildMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildMessageReactions)

type GalactusAPI struct {
	client        *redis.Client
	storageClient *storage.PsqlInterface
	shardManager  *dshardmanager.Manager
	topggClient   *dbl.Client
	botID         string

	// maps hashed tokens to active discord sessions
	activeSessions      map[string]*discordgo.Session
	maxRequests5Seconds int64
	sessionLock         sync.RWMutex

	logger *zap.Logger
}

func NewGalactusAPI(logger *zap.Logger, botToken string, numShards int, topGGtoken, botID, redisAddr, redisUser, redisPass string, maxReq int64, botPrefix string) *GalactusAPI {
	var rdb *redis.Client

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPass,
		DB:       0, // use default DB
	})

	manager := shard_manager.MakeShardManager(logger, botToken, numShards, DefaultIntents)
	shard_manager.AddHandlers(logger, manager, rdb, botPrefix)
	shard_manager.AddRateLimitHandler(manager, RateLimitHandler(logger, rdb))

	var topgg *dbl.Client = nil
	if topGGtoken != "" {
		topgg, _ = dbl.NewClient(topGGtoken)
	}

	return &GalactusAPI{
		client:              rdb,
		shardManager:        manager,
		topggClient:         topgg,
		botID:               botID,
		activeSessions:      make(map[string]*discordgo.Session),
		maxRequests5Seconds: maxReq,
		sessionLock:         sync.RWMutex{},
		logger:              logger,
	}
}

func (galactus *GalactusAPI) InitStorage(addr, user, pass string) error {
	connectUrl := storage.ConstructPsqlConnectURL(addr, user, pass)
	var storageClient = storage.PsqlInterface{}
	err := storageClient.Init(connectUrl)
	if err != nil {
		return err
	}
	galactus.storageClient = &storageClient
	return nil
}

func (galactus *GalactusAPI) getAllTokensForGuild(guildID string) []string {
	hTokens, err := galactus.client.SMembers(context.Background(), rediskey.GuildTokensKey(guildID)).Result()
	if err != nil {
		galactus.logger.Error("error retrieving smembers from Redis",
			zap.Error(err),
			zap.String("guildID", guildID),
		)
		return nil
	}
	return hTokens
}

func (galactus *GalactusAPI) getAnySession(guildID string, tokens []string, limit int) (*discordgo.Session, string) {
	galactus.sessionLock.RLock()
	defer galactus.sessionLock.RUnlock()

	for i, hToken := range tokens {
		if i == limit {
			return nil, ""
		}
		// if this token isn't potentially rate-limited
		if galactus.IncrAndTestGuildTokenComboLock(guildID, hToken) {
			sess, ok := galactus.activeSessions[hToken]
			if ok {
				return sess, hToken
			}
			// remove this key from our records and keep going
			galactus.client.SRem(context.Background(), rediskey.GuildTokensKey(guildID), hToken)
		} else {
			galactus.logger.Info("secondary token potentially rate-limited; skipping",
				zap.String("hashedToken", hToken),
				zap.String("guildID", guildID),
			)
		}
	}

	return nil, ""
}

func (galactus *GalactusAPI) IncrAndTestGuildTokenComboLock(guildID, hashToken string) bool {
	i, err := galactus.client.Incr(context.Background(), rediskey.GuildTokenLock(guildID, hashToken)).Result()
	if err != nil {
		galactus.logger.Error("error incrementing guild token combo",
			zap.Error(err),
			zap.String("guildID", guildID),
			zap.String("hashedToken", hashToken),
		)
	}
	usable := i < galactus.maxRequests5Seconds
	galactus.logger.Info("guild token combo",
		zap.String("guildID", guildID),
		zap.String("hashedToken", hashToken),
		zap.Int64("count", i),
		zap.Bool("using", usable),
	)
	if !usable {
		return false
	}

	err = galactus.client.Expire(context.Background(), rediskey.GuildTokenLock(guildID, hashToken), time.Second*5).Err()
	if err != nil {
		galactus.logger.Error("error setting expiration for guild token combo",
			zap.Error(err),
			zap.String("guildID", guildID),
			zap.String("hashedToken", hashToken),
		)
	}

	return true
}

func (galactus *GalactusAPI) BlacklistTokenForDuration(guildID, hashToken string, duration time.Duration) error {
	return galactus.client.Set(context.Background(), rediskey.GuildTokenLock(guildID, hashToken), galactus.maxRequests5Seconds, duration).Err()
}

type JobsNumber struct {
	Jobs int64 `json:"jobs"`
}

func (galactus *GalactusAPI) Run(port string, maxWorkers int, captureAckTimeout time.Duration, taskTimeout time.Duration) {

	galactus.loadTokensFromEnv()

	go PrometheusMetricsServer(galactus.client, "2112")

	// TODO maybe eventually provide some auth parameter, or version number? Something to prove that a worker can pop requests?
	mainRouter := mux.NewRouter()

	generalRouter := mainRouter.PathPrefix(endpoint.GeneralRoute).Subrouter()
	captureRouter := mainRouter.PathPrefix(endpoint.CaptureRoute).Subrouter()
	discordRouter := mainRouter.PathPrefix(endpoint.DiscordRoute).Subrouter()
	settingsRouter := mainRouter.PathPrefix(endpoint.SettingsRoute).Subrouter()

	generalRouter.HandleFunc("/", galactus.indexHandler()).Methods("GET")

	discordRouter.HandleFunc(endpoint.DiscordJobCount, galactus.jobCount()).Methods("GET")
	discordRouter.HandleFunc(endpoint.DiscordJobRequest, galactus.requestJobHandler(taskTimeout)).Methods("POST")
	discordRouter.HandleFunc(endpoint.ModifyUserFull, galactus.modifyUserHandler(maxWorkers, captureAckTimeout)).Methods("POST")
	discordRouter.HandleFunc(endpoint.SendMessageFull, galactus.SendChannelMessageHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.SendMessageEmbedFull, galactus.SendChannelMessageEmbedHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.EditMessageEmbedFull, galactus.EditMessageEmbedHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.DeleteMessageFull, galactus.DeleteChannelMessageHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildFull, galactus.GetGuildHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildChannelsFull, galactus.GetGuildChannelsHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildMemberFull, galactus.GetGuildMemberHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildRolesFull, galactus.GetGuildRolesHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.AddReactionFull, galactus.AddReactionHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.RemoveReactionFull, galactus.RemoveReactionHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.RemoveAllReactionsFull, galactus.RemoveAllReactionsHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.UserChannelCreateFull, galactus.CreateUserChannelHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildEmojisFull, galactus.GetGuildEmojisHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.GetGuildPremiumFull, galactus.GetGuildPremiumHandler()).Methods("POST")
	discordRouter.HandleFunc(endpoint.CreateGuildEmojiFull, galactus.CreateGuildEmojiHandler()).Methods("POST")

	settingsRouter.HandleFunc(endpoint.GetGuildAMUSettingsFull, galactus.GetGuildAMUSettings()).Methods("POST")

	captureRouter.HandleFunc(endpoint.GetCaptureTaskFull, galactus.GetCaptureTaskHandler(taskTimeout)).Methods("POST")
	captureRouter.HandleFunc(endpoint.SetCaptureTaskStatusFull, galactus.SetCaptureTaskStatusHandler()).Methods("POST")
	captureRouter.HandleFunc(endpoint.AddCaptureEventFull, galactus.AddCaptureEventHandler()).Methods("POST")
	captureRouter.HandleFunc(endpoint.GetCaptureEventFull, galactus.GetCaptureEventHandler(taskTimeout)).Methods("POST")

	galactus.logger.Info("galactus is running",
		zap.String("port", port),
	)

	err := http.ListenAndServe(":"+port, mainRouter)
	if err != nil {
		galactus.logger.Error("http listener exited with error",
			zap.Error(err),
		)
	}
}

func (galactus *GalactusAPI) indexHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO For any higher-sensitivity info in the future, this should properly identify the origin specifically
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length")

		// default to listing active games in the last 15 mins
		activeGames := rediskey.GetActiveGames(context.Background(), galactus.client, 900)
		version, commit := rediskey.GetVersionAndCommit(context.Background(), galactus.client)
		totalGuilds := rediskey.GetGuildCounter(context.Background(), galactus.client)
		totalUsers := rediskey.GetTotalUsers(context.Background(), galactus.client)
		totalGames := rediskey.GetTotalGames(context.Background(), galactus.client)

		data := map[string]interface{}{
			"version":     version,
			"commit":      commit,
			"totalGuilds": totalGuilds,
			"activeGames": activeGames,
			"totalUsers":  totalUsers,
			"totalGames":  totalGames,
		}

		w.WriteHeader(http.StatusOK)
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			galactus.logger.Error("error marshalling data for index endpoint",
				zap.Error(err),
			)
		} else {
			w.Write(jsonBytes)
		}
	}
}

func (galactus *GalactusAPI) requestJobHandler(timeout time.Duration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		msg, err := redisutils.PopRawDiscordMessageTimeout(galactus.client, timeout)

		// no jobs available
		switch {
		case errors.Is(err, redis.Nil):
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("{\"status\": \"No jobs available\"}"))
			return
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
			galactus.logger.Error("redis error when popping job",
				zap.String("endpoint", endpoint.DiscordJobRequest),
				zap.Error(err))
			return
		case msg == "":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"Nil job returned, despite no Redis errors\"}"))
			galactus.logger.Error("nil job returned, despite no Redis errors",
				zap.String("endpoint", endpoint.DiscordJobRequest))
			return
		}

		w.WriteHeader(http.StatusOK)

		_, err = w.Write([]byte(msg))
		if err != nil {
			galactus.logger.Error("failed to write job as HTTP response",
				zap.String("endpoint", endpoint.DiscordJobRequest),
				zap.Error(err),
			)
		}
	}
}

func (galactus *GalactusAPI) jobCount() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobs JobsNumber

		num, err := redisutils.DiscordMessagesSize(galactus.client)
		if err == nil || errors.Is(err, redis.Nil) {
			if errors.Is(err, redis.Nil) {
				jobs.Jobs = 0
			} else {
				jobs.Jobs = num
			}

			byt, err := json.Marshal(jobs)
			if err != nil {
				galactus.logger.Error("error marshalling JobsNumber",
					zap.Error(err),
				)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write(byt)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
		}
	}
}

func (galactus *GalactusAPI) loadTokensFromEnv() {
	workerTokenStr := strings.ReplaceAll(os.Getenv("WORKER_BOT_TOKENS"), " ", "")
	if workerTokenStr == "" {
		galactus.logger.Info("no WORKER_BOT_TOKENS provided")
		return
	}
	botTokens := strings.Split(workerTokenStr, ",")
	for _, botToken := range botTokens {
		hashedToken := hashToken(botToken)
		galactus.logger.Info("loaded bot token",
			zap.String("token", botToken))

		galactus.sessionLock.RLock()
		if _, ok := galactus.activeSessions[hashedToken]; ok {
			galactus.logger.Info("token already has a running session on this instance",
				zap.String("token", botToken))
			galactus.sessionLock.RUnlock()
			continue
		}
		galactus.sessionLock.RUnlock()

		token.WaitForToken(galactus.client, botToken)
		token.LockForToken(galactus.client, botToken)

		sess, err := discordgo.New("Bot " + botToken)
		if err != nil {
			galactus.logger.Error("error in CREATING discordgo session, possibly an invalid token",
				zap.Error(err),
				zap.String("token", botToken))
			continue
		}
		sess.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuilds)
		sess.AddHandler(galactus.newGuildHandler(hashedToken))
		err = sess.Open()
		if err != nil {
			galactus.logger.Error("error in OPENING discordgo session, possibly an invalid token",
				zap.Error(err),
				zap.String("token", botToken))
			continue
		}

		galactus.sessionLock.Lock()
		galactus.activeSessions[hashedToken] = sess
		galactus.sessionLock.Unlock()

		for _, v := range sess.State.Guilds {
			err := galactus.client.SAdd(context.Background(), rediskey.GuildTokensKey(v.ID), hashedToken).Err()
			if !errors.Is(err, redis.Nil) && err != nil {
				galactus.logger.Error("error adding bot token for specific guild",
					zap.Error(err),
					zap.String("token", botToken),
					zap.String("guildID", v.ID))
			} else {
				galactus.logger.Info("added bot token to guild successfully",
					zap.String("token", botToken),
					zap.String("guildID", v.ID),
				)
			}
		}
	}
}

func (galactus *GalactusAPI) waitForAck(pubsub *redis.PubSub, waitTime time.Duration, result chan<- bool) {
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

func (galactus *GalactusAPI) Close() {
	err := galactus.shardManager.StopAll()
	if err != nil {
		galactus.logger.Error("error stopping all shard sessions",
			zap.Error(err),
		)
	}

	galactus.sessionLock.Lock()
	for hToken, v := range galactus.activeSessions {
		err = v.Close()
		if err != nil {
			galactus.logger.Error("error closing active session",
				zap.Error(err),
				zap.String("hashedToken", hToken),
			)
		}
	}
	galactus.activeSessions = map[string]*discordgo.Session{}
	galactus.sessionLock.Unlock()

	if galactus.storageClient != nil {
		galactus.storageClient.Close()
	}
}

func (galactus *GalactusAPI) newGuildHandler(hashedToken string) func(s *discordgo.Session, m *discordgo.GuildCreate) {
	return func(s *discordgo.Session, m *discordgo.GuildCreate) {
		galactus.sessionLock.RLock()
		for test := range galactus.activeSessions {
			if hashedToken == test {
				err := galactus.client.SAdd(context.Background(), rediskey.GuildTokensKey(m.Guild.ID), hashedToken).Err()
				if err != nil {
					galactus.logger.Error("error adding hashed token for guild",
						zap.Error(err),
						zap.String("hashedToken", hashedToken),
						zap.String("guildID", m.Guild.ID),
					)
				} else {
					galactus.logger.Info("token added for guild",
						zap.String("guildID", m.Guild.ID),
					)
				}
			}
		}

		galactus.sessionLock.RUnlock()
	}
}
