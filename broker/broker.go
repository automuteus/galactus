package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/automuteus/galactus/discord"
	"github.com/go-redis/redis/v8"
	socketio "github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const ConnectCodeLength = 8

var ctx = context.Background()

func activeGamesCode() string {
	return "automuteus:games"
}

type GameLobby struct {
	LobbyCode string `json:"LobbyCode"`
	Region    int    `json:"Region"`
	PlayMap   int    `json:"Map"`
}

func roomCodesForConnCodeKey(connCode string) string {
	return "automuteus:roomcode:" + connCode
}

type Broker struct {
	client *redis.Client

	//map of socket IDs to connection codes
	connections map[string]string

	ackKillChannels map[string]chan bool
	connectionsLock sync.RWMutex
}

func NewBroker(redisAddr, redisUser, redisPass string) *Broker {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Username: redisUser,
		Password: redisPass,
		DB:       0, // use default DB
	})
	return &Broker{
		client:          rdb,
		connections:     map[string]string{},
		ackKillChannels: map[string]chan bool{},
		connectionsLock: sync.RWMutex{},
	}
}

func (broker *Broker) TasksListener(server *socketio.Server, connectCode string, killchan <-chan bool) {
	pubsub := broker.client.Subscribe(context.Background(), discord.TasksSubscribeKey(connectCode))
	log.Println("Task listener OPEN for " + connectCode)
	defer log.Println("Task listener CLOSE for " + connectCode)
	channel := pubsub.Channel()
	for {
		select {
		case t := <-channel:
			task := discord.ModifyTask{}

			err := json.Unmarshal([]byte(t.Payload), &task)
			if err != nil {
				log.Println(err)
				break
			}

			log.Println("Broadcasting " + t.Payload + " to room " + connectCode)
			server.BroadcastToRoom("/", connectCode, "modify", t.Payload)
			break
		case <-killchan:
			pubsub.Close()
			return
		}
	}
}

func (broker *Broker) Start(port string) {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		log.Println("connected:", s.ID())
		return nil
	})
	server.OnEvent("/", "connectCode", func(s socketio.Conn, msg string) {
		log.Printf("Received connection code: \"%s\"", msg)

		if len(msg) != ConnectCodeLength {
			s.Close()
		} else {
			killChannel := make(chan bool)

			broker.connectionsLock.Lock()
			broker.connections[s.ID()] = msg
			broker.ackKillChannels[s.ID()] = killChannel
			broker.connectionsLock.Unlock()

			err := PushJob(ctx, broker.client, msg, Connection, "true")
			if err != nil {
				log.Println(err)
			}
			go broker.AckWorker(ctx, msg, killChannel)
		}
	})

	//only join the room for the connect code once we ensure that the bot actually connects with a valid discord session
	server.OnEvent("/", "botID", func(s socketio.Conn, msg int64) {
		log.Printf("Received bot ID: \"%d\"", msg)

		broker.connectionsLock.RLock()
		if code, ok := broker.connections[s.ID()]; ok {
			//this socket is now listening for mutes that can be applied via that connect code
			s.Join(code)
			killChan := broker.ackKillChannels[s.ID()]
			if killChan != nil {
				go broker.TasksListener(server, code, killChan)
			} else {
				log.Println("Null killchannel for conncode: " + code + ". This means we got a Bot ID before a connect code!")
			}
		}
		broker.connectionsLock.RUnlock()
	})

	server.OnEvent("/", "taskFailed", func(s socketio.Conn, msg string) {
		log.Printf("Received failure for task ID: \"%s\"", msg)

		broker.client.Publish(context.Background(), discord.CompleteTaskAckKey(msg), "false")
	})

	server.OnEvent("/", "taskComplete", func(s socketio.Conn, msg string) {
		log.Printf("Received success for task ID: \"%s\"", msg)

		broker.client.Publish(context.Background(), discord.CompleteTaskAckKey(msg), "true")
	})

	server.OnEvent("/", "lobby", func(s socketio.Conn, msg string) {
		log.Println("lobby:", msg)

		//validation
		var lobby GameLobby
		err := json.Unmarshal([]byte(msg), &lobby)
		if err != nil {
			log.Println(err)
		} else {
			broker.connectionsLock.RLock()
			if cCode, ok := broker.connections[s.ID()]; ok {
				err := PushJob(ctx, broker.client, cCode, Lobby, msg)
				if err != nil {
					log.Println(err)
				}
				err = broker.client.Set(context.Background(), roomCodesForConnCodeKey(cCode), lobby.LobbyCode, time.Minute*15).Err()
				if err != nil {
					log.Println(err)
				} else {
					log.Printf("Updated room code %s for connect code %s in Redis", lobby.LobbyCode, cCode)
				}
			}
			broker.connectionsLock.RUnlock()
		}
	})
	server.OnEvent("/", "state", func(s socketio.Conn, msg string) {
		log.Println("phase received from capture: ", msg)
		_, err := strconv.Atoi(msg)
		if err != nil {
			log.Println(err)
		} else {
			broker.connectionsLock.RLock()
			if cCode, ok := broker.connections[s.ID()]; ok {
				err := PushJob(ctx, broker.client, cCode, State, msg)
				if err != nil {
					log.Println(err)
				}
				err = broker.client.Expire(context.Background(), roomCodesForConnCodeKey(cCode), time.Minute*15).Err()
				if err != redis.Nil && err != nil {
					log.Println(err)
				}
			}
			broker.connectionsLock.RUnlock()
		}
	})
	server.OnEvent("/", "player", func(s socketio.Conn, msg string) {
		log.Println("player received from capture: ", msg)

		broker.connectionsLock.RLock()
		if cCode, ok := broker.connections[s.ID()]; ok {
			err := PushJob(ctx, broker.client, cCode, Player, msg)
			if err != nil {
				log.Println(err)
			}
			err = broker.client.Expire(context.Background(), roomCodesForConnCodeKey(cCode), time.Minute*15).Err()
			if err != redis.Nil && err != nil {
				log.Println(err)
			}
		}
		broker.connectionsLock.RUnlock()
	})
	server.OnEvent("/", "gameover", func(s socketio.Conn, msg string) {
		broker.connectionsLock.RLock()
		if cCode, ok := broker.connections[s.ID()]; ok {
			err := PushJob(ctx, broker.client, cCode, GameOver, msg)
			if err != nil {
				log.Println(err)
			}
		}
		broker.connectionsLock.RUnlock()
	})
	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("Client connection closed: ", reason)

		broker.connectionsLock.RLock()
		if cCode, ok := broker.connections[s.ID()]; ok {
			err := PushJob(ctx, broker.client, cCode, Connection, "false")
			if err != nil {
				log.Println(err)
			}
			server.ClearRoom("/", cCode)
		}
		broker.connectionsLock.RUnlock()

		broker.connectionsLock.Lock()
		if c, ok := broker.ackKillChannels[s.ID()]; ok {
			c <- true
		}
		delete(broker.ackKillChannels, s.ID())
		delete(broker.connections, s.ID())
		broker.connectionsLock.Unlock()
	})
	go server.Serve()
	defer server.Close()

	router := mux.NewRouter()
	router.Handle("/socket.io/", server)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		broker.connectionsLock.RLock()
		activeConns := len(broker.connections)
		broker.connectionsLock.RUnlock()

		//default to listing active games in the last 15 mins
		activeGames := GetActiveGames(broker.client, 900)
		version, commit := GetVersionAndCommit(broker.client)
		totalGuilds := GetGuildCounter(broker.client)

		data := map[string]interface{}{
			"version":           version,
			"commit":            commit,
			"totalGuilds":       totalGuilds,
			"activeConnections": activeConns,
			"activeGames":       activeGames,
		}

		jsonBytes, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
		}
		w.Write(jsonBytes)
	})

	router.HandleFunc("/lobbycode/{connectCode}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		conncode := vars["connectCode"]

		if conncode == "" || len(conncode) != ConnectCodeLength {
			errorResponse(w)
			return
		}

		key, err := broker.client.Get(context.Background(), roomCodesForConnCodeKey(conncode)).Result()
		if err == redis.Nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := Resp{Result: key}
		jbytes, err := json.Marshal(resp)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jbytes)
		}
		return
	})
	log.Printf("Message broker is running on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

type Resp struct {
	Result string `json:"result"`
}

func errorResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
	r := Resp{Result: "error"}
	jbytes, err := json.Marshal(r)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.Write(jbytes)
	}
	return
}

func totalGuildsKey() string {
	return "automuteus:count:guilds"
}

//TODO these are duplicated in the main repo and here! Eek!
func versionKey() string {
	return "automuteus:version"
}

func commitKey() string {
	return "automuteus:commit"
}

///////

func GetVersionAndCommit(client *redis.Client) (string, string) {
	v, err := client.Get(ctx, versionKey()).Result()
	if err != nil {
		log.Println(err)
	}
	c, err := client.Get(ctx, commitKey()).Result()
	if err != nil {
		log.Println(err)
	}
	return v, c
}

func GetGuildCounter(client *redis.Client) int64 {
	count, err := client.SCard(ctx, totalGuildsKey()).Result()
	if err != nil {
		log.Println(err)
		return 0
	}
	return count
}

func GetActiveGames(client *redis.Client, secs int64) int64 {
	now := time.Now()
	before := now.Add(-(time.Second * time.Duration(secs)))
	count, err := client.ZCount(ctx, activeGamesCode(), fmt.Sprintf("%d", before.Unix()), fmt.Sprintf("%d", now.Unix())).Result()
	if err != nil {
		log.Println(err)
		return 0
	}
	return count
}

func RemoveActiveGame(client *redis.Client, connectCode string) {
	client.ZRem(ctx, activeGamesCode(), connectCode)
}

//anytime a bot "acks", then push a notification
func (broker *Broker) AckWorker(ctx context.Context, connCode string, killChan <-chan bool) {
	pubsub := AckSubscribe(ctx, broker.client, connCode)
	channel := pubsub.Channel()
	defer pubsub.Close()

	for {
		select {
		case <-killChan:
			return
		case <-channel:
			err := PushJob(ctx, broker.client, connCode, Connection, "true")
			if err != nil {
				log.Println(err)
			}
			//notify(ctx, broker.client, connCode)
			break
		}
	}
}
