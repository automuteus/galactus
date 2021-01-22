package main

import (
	"github.com/automuteus/galactus/internal/galactus"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const DefaultGalactusPort = "5858"

const DefaultMaxRequests5Sec int64 = 7
const DefaultMaxWorkers = 8
const DefaultCaptureBotTimeout = time.Second
const DefaultTaskTimeout = time.Second * 10
const DefaultBotPrefix = ".au"

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

	maxReq := DefaultMaxRequests5Sec
	maxReq5Sec := os.Getenv("MAX_REQ_5_SEC")
	if maxReq5Sec != "" {
		num, err := strconv.ParseInt(maxReq5Sec, 10, 64)
		if err == nil {
			maxReq = num
		} else {
			logger.Error("failed to parse MAX_REQ_5_SEC as int64",
				zap.String("received", maxReq5Sec))
		}
	}

	captureAckTimeout := DefaultCaptureBotTimeout
	captureAckTimeoutStr := os.Getenv("ACK_TIMEOUT_MS")
	num, err := strconv.ParseInt(captureAckTimeoutStr, 10, 64)
	if err == nil {
		captureAckTimeout = time.Millisecond * time.Duration(num)
	} else {
		logger.Error("could not parse ACK_TIMEOUT_MS",
			zap.Error(err),
			zap.Int64("default", captureAckTimeout.Milliseconds()))
	}

	taskTimeout := DefaultTaskTimeout
	taskTimeoutStr := os.Getenv("TASK_TIMEOUT_MS")
	num, err = strconv.ParseInt(taskTimeoutStr, 10, 64)
	if err == nil {
		taskTimeout = time.Millisecond * time.Duration(num)
	} else {
		logger.Error("could not parse TASK_TIMEOUT_MS",
			zap.Error(err),
			zap.Int64("default", taskTimeout.Milliseconds()))
	}

	maxWorkers := DefaultMaxWorkers
	maxWorkersStr := os.Getenv("MAX_WORKERS")
	num, err = strconv.ParseInt(maxWorkersStr, 10, 64)
	if err == nil {
		maxWorkers = int(num)
	} else {
		logger.Error("could not parse MAX_WORKERS",
			zap.Error(err),
			zap.Int("default", maxWorkers))
	}

	botPrefix := DefaultBotPrefix
	if os.Getenv("AUTOMUTEUS_GLOBAL_PREFIX") != "" {
		botPrefix = os.Getenv("AUTOMUTEUS_GLOBAL_PREFIX")
	}

	logger.Info("loaded env",
		zap.String("DISCORD_BOT_TOKEN", botToken),
		zap.String("REDIS_ADDR", redisAddr),
		zap.String("REDIS_USER", redisUser),
		zap.String("REDIS_PASS", redisPass),
		zap.Int("MAX_REQ_5_SEC", int(maxReq)),
		zap.Int("MAX_WORKERS", maxWorkers),
		zap.Int64("ACK_TIMEOUT_MS", captureAckTimeout.Milliseconds()),
		zap.String("AUTOMUTEUS_GLOBAL_PREFIX", botPrefix),
	)

	tp := galactus.NewGalactusAPI(logger, botToken, redisAddr, redisUser, redisPass, maxReq, botToken)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	go tp.Run(galactusPort, maxWorkers, captureAckTimeout, taskTimeout)
	<-sc
	tp.Close()
}
