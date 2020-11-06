package main

import (
	"github.com/automuteus/galactus/galactus"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

const DefaultPort = "5858"

func main() {
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		log.Fatal("No REDIS_ADDRESS specified. Exiting.")
	}

	port := os.Getenv("GALACTUS_PORT")
	if port == "" {
		log.Println("No GALACTUS_PORT provided. Defaulting to " + DefaultPort)
		port = DefaultPort
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

	redisDB := os.Getenv("REDIS_DB")
	num, err := strconv.ParseInt(redisDB, 10, 64)
	if err != nil {
		log.Println("Invalid REDIS_DB provided. Defaulting to 0")
		num = 0
	}

	tp := galactus.NewTokenProvider(redisAddr, redisUser, redisPass, int(num))

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go tp.Run(port)
	<-sc
	tp.Close()
}
