package main

import (
	"github.com/automuteus/galactus/broker"
	"github.com/automuteus/galactus/galactus"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

const DefaultGalactusPort = "5858"
const DefaultBrokerPort = "8123"
const DefaultMaxRequests5Sec int64 = 7

func main() {
	botToken := os.Getenv("DISCORD_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("No DISCORD_BOT_TOKEN specified. Exiting.")
	}

	var extraTokens []string
	extraTokenStr := strings.ReplaceAll(os.Getenv("WORKER_BOT_TOKENS"), " ", "")
	if extraTokenStr != "" {
		extraTokens = strings.Split(extraTokenStr, ",")
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

	maxReq5Sec := os.Getenv("MAX_REQ_5_SEC")
	maxReq := DefaultMaxRequests5Sec
	num, err := strconv.ParseInt(maxReq5Sec, 10, 64)
	if err == nil {
		maxReq = num
	}

	tp := galactus.NewTokenProvider(botToken, redisAddr, redisUser, redisPass, maxReq)
	tp.PopulateAndStartSessions(extraTokens)
	msgBroker := broker.NewBroker(redisAddr, redisUser, redisPass)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	go msgBroker.Start(brokerPort)

	go tp.Run(galactusPort)
	<-sc
	tp.Close()
}
