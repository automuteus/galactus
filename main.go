package main

import (
	"github.com/automuteus/galactus/broker"
	"github.com/automuteus/galactus/galactus"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const DefaultGalactusPort = "5858"
const DefaultBrokerPort = "8123"

func main() {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("No DISCORD_BOT_TOKEN specified. Exiting.")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("No REDIS_ADDR specified. Exiting.")
	}

	galactusPort := os.Getenv("GALACTUS_PORT")
	if galactusPort == "" {
		log.Println("No GALACTUS_PORT provided. Defaulting to " + DefaultGalactusPort)
		galactusPort = DefaultGalactusPort
	}
	brokerPort := os.Getenv("BROKER_PORT")
	if brokerPort == "" {
		log.Println("No BROKER_PORT provided. Defaulting to " + DefaultBrokerPort)
		brokerPort = DefaultBrokerPort
	}

	redisUser := os.Getenv("REDIS_USER")
	redisPass := os.Getenv("REDIS_PASS")
	if redisUser != "" {
		log.Println("Using REDIS_USER=" + redisUser)
	} else {
		log.Println("No REDIS_USER specified.")
	}

	if redisPass != "" {
		log.Println("Using REDIS_PASS=<redacted>")
	} else {
		log.Println("No REDIS_PASS specified.")
	}

	tp := galactus.NewTokenProvider(botToken, redisAddr, redisUser, redisPass)
	tp.PopulateAndStartSessions()
	msgBroker := broker.NewBroker(redisAddr, redisUser, redisPass)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go msgBroker.Start(brokerPort)

	go tp.Run(galactusPort)
	<-sc
	tp.Close()
}
