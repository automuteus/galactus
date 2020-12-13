package broker

import (
	"context"
	"encoding/json"
	"github.com/automuteus/utils/pkg/game"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/task"
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
	pubsub := broker.client.Subscribe(context.Background(), rediskey.TasksSubscribe(connectCode))
	log.Println("Task listener OPEN for " + connectCode)
	defer log.Println("Task listener CLOSE for " + connectCode)
	channel := pubsub.Channel()
	for {
		select {
		case t := <-channel:
			taskObj := task.ModifyTask{}

			err := json.Unmarshal([]byte(t.Payload), &taskObj)
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

			err := task.PushJob(context.Background(), broker.client, msg, task.ConnectionJob, "true")
			if err != nil {
				log.Println(err)
			}
			go broker.AckWorker(context.Background(), msg, killChannel)
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

		broker.client.Publish(context.Background(), rediskey.CompleteTask(msg), "false")
	})

	server.OnEvent("/", "taskComplete", func(s socketio.Conn, msg string) {
		log.Printf("Received success for task ID: \"%s\"", msg)

		broker.client.Publish(context.Background(), rediskey.CompleteTask(msg), "true")
	})

	server.OnEvent("/", "lobby", func(s socketio.Conn, msg string) {
		log.Println("lobby:", msg)

		//validation
		var lobby game.Lobby
		err := json.Unmarshal([]byte(msg), &lobby)
		if err != nil {
			log.Println(err)
		} else {
			broker.connectionsLock.RLock()
			if cCode, ok := broker.connections[s.ID()]; ok {
				err := task.PushJob(context.Background(), broker.client, cCode, task.LobbyJob, msg)
				if err != nil {
					log.Println(err)
				}
				err = broker.client.Set(context.Background(), rediskey.RoomCodesForConnCode(cCode), lobby.LobbyCode, time.Minute*15).Err()
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
				err := task.PushJob(context.Background(), broker.client, cCode, task.StateJob, msg)
				if err != nil {
					log.Println(err)
				}
				err = broker.client.Expire(context.Background(), rediskey.RoomCodesForConnCode(cCode), time.Minute*15).Err()
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
			err := task.PushJob(context.Background(), broker.client, cCode, task.PlayerJob, msg)
			if err != nil {
				log.Println(err)
			}
			err = broker.client.Expire(context.Background(), rediskey.RoomCodesForConnCode(cCode), time.Minute*15).Err()
			if err != redis.Nil && err != nil {
				log.Println(err)
			}
		}
		broker.connectionsLock.RUnlock()
	})
	server.OnEvent("/", "gameover", func(s socketio.Conn, msg string) {
		broker.connectionsLock.RLock()
		if cCode, ok := broker.connections[s.ID()]; ok {
			err := task.PushJob(context.Background(), broker.client, cCode, task.GameOverJob, msg)
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
			err := task.PushJob(context.Background(), broker.client, cCode, task.ConnectionJob, "false")
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
		activeGames := rediskey.GetActiveGames(context.Background(), broker.client, 900)
		version, commit := rediskey.GetVersionAndCommit(context.Background(), broker.client)
		totalGuilds := rediskey.GetGuildCounter(context.Background(), broker.client)

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

		key, err := broker.client.Get(context.Background(), rediskey.RoomCodesForConnCode(conncode)).Result()
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

//anytime a bot "acks", then push a notification
func (broker *Broker) AckWorker(ctx context.Context, connCode string, killChan <-chan bool) {
	pubsub := task.AckSubscribe(ctx, broker.client, connCode)
	channel := pubsub.Channel()
	defer pubsub.Close()

	for {
		select {
		case <-killChan:
			return
		case <-channel:
			err := task.PushJob(ctx, broker.client, connCode, task.ConnectionJob, "true")
			if err != nil {
				log.Println(err)
			}
			break
		}
	}
}
