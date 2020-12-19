package internal

import (
	"github.com/automuteus/galactus/internal/handler"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/jonas747/dshardmanager"
	"log"
)

const DefaultShards = 10

func MakeShardManager(token string, intent *discordgo.Intent) *dshardmanager.Manager {
	manager := dshardmanager.New("Bot " + token)
	manager.Name = "AutoMuteUs"

	recommended, err := manager.GetRecommendedCount()
	if err != nil {
		log.Fatal("Failed getting recommended shard count")
	}
	if recommended < 2 {
		manager.SetNumShards(DefaultShards)
	}

	log.Println("Starting the shard manager")
	err = manager.Start()
	if err != nil {
		log.Fatal("Failed to start: ", err)
	}

	log.Println("Started!")

	manager.Lock()
	for _, v := range manager.Sessions {
		v.Identify.Intents = intent
	}
	manager.Unlock()

	return manager
}

func AddHandlers(manager *dshardmanager.Manager, client *redis.Client) {
	manager.AddHandler(handler.GuildCreateHandler(client))
	manager.AddHandler(handler.GuildDeleteHandler(client))

	manager.AddHandler(handler.VoiceStateUpdateHandler(client))
	manager.AddHandler(handler.MessageCreateHandler(client))
	manager.AddHandler(handler.MessageReactionAddHandler(client))
}
