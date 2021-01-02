package main

import (
	"github.com/automuteus/galactus/internal"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

const DefaultGalactusPort = "5858"
const DefaultMaxRequests5Sec int64 = 7

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Println("Failed to initialize logger with error")
		log.Fatal(err)
	}

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

	redisUser := os.Getenv("REDIS_USER")
	redisPass := os.Getenv("REDIS_PASS")

	maxReq5Sec := os.Getenv("MAX_REQ_5_SEC")
	maxReq := DefaultMaxRequests5Sec
	if maxReq5Sec != "" {
		num, err := strconv.ParseInt(maxReq5Sec, 10, 64)
		if err == nil {
			maxReq = num
		} else {
			logger.Error("failed to parse MAX_REQ_5_SEC as int64",
				zap.String("received", maxReq5Sec))
		}
	}

	logger.Info("loaded env",
		zap.String("DISCORD_BOT_TOKEN", botToken),
		zap.String("REDIS_ADDR", redisAddr),
		zap.String("REDIS_USER", redisUser),
		zap.String("REDIS_PASS", redisPass),
		zap.Int("MAX_REQ_5_SEC", int(maxReq)),
	)

	tp := internal.NewGalactusAPI(logger, botToken, redisAddr, redisUser, redisPass, maxReq)
	tp.PopulateAndStartSessions()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go tp.Run(logger, galactusPort)
	<-sc
	tp.Close()
}
