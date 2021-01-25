module github.com/automuteus/galactus

go 1.15

require (
	github.com/alicebob/miniredis/v2 v2.14.1
	github.com/automuteus/utils v0.0.10
	github.com/bsm/redislock v0.7.0
	github.com/bwmarrin/discordgo v0.22.1
	github.com/go-redis/redis/v8 v8.4.2
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/jonas747/dshardmanager v0.0.0-20180911185241-9e4282faed43
	github.com/top-gg/go-dbl v0.0.0-20201116001615-e844586b1159
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
)

// TODO replace when V7 comes out
replace github.com/automuteus/utils v0.0.10 => github.com/automuteus/utils v0.0.11-0.20210117090606-d48d8a0c6a4b
