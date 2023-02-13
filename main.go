package main

import (
	"github.com/automuteus/galactus/broker"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const DefaultBrokerPort = "8123"

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("No REDIS_ADDR specified. Exiting.")
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

	msgBroker := broker.NewBroker(redisAddr, redisUser, redisPass)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	go msgBroker.Start(brokerPort)
	<-sc
}
