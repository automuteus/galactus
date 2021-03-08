package galactus

import (
	"context"
	"errors"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"
)

type EventType int

const (
	MuteDeafenOfficial EventType = iota
	MuteDeafenCapture
	MuteDeafenWorker
	MessageCreate
	MessageEmbedCreate
	MessageDelete
	MessageEmbedEdit
	ReactionAdd
	ReactionRemove
	ReactionRemoveAll
	Guild
	GuildChannels
	GuildEmojis
	GuildMember
	GuildRoles
	CreateGuildEmoji
	UserChannel
	InvalidRequest
	OfficialRequest //must be the last metric
)

var MetricTypeStrings = []string{
	"mute_deafen_official",
	"mute_deafen_capture",
	"mute_deafen_worker",
	"message_create",
	"message_embed_create",
	"message_delete",
	"message_embed_edit",
	"reaction_add",
	"reaction_remove",
	"reaction_remove_all",
	"guild",
	"guild_channels",
	"guild_emojis",
	"guild_member",
	"guild_roles",
	"create_guild_emoji",
	"user_channel",
	"invalid_request",
	"official_request", //must be the last request, because of how the sum is calculated in Collect below
}

type Collector struct {
	counterDesc *prometheus.Desc
	client      *redis.Client
	commit      string
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.counterDesc
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	official := int64(0)
	for i, str := range MetricTypeStrings {
		if i != int(OfficialRequest) {
			v, err := c.client.Get(context.Background(), rediskey.RequestsByType(str)).Result()
			if !errors.Is(err, redis.Nil) && err != nil {
				log.Println(err)
				continue
			} else {
				num := int64(0)
				if v != "" {
					num, err = strconv.ParseInt(v, 10, 64)
					if err != nil {
						log.Println(err)
						num = 0
					}
				}

				ch <- prometheus.MustNewConstMetric(
					c.counterDesc,
					prometheus.CounterValue,
					float64(num),
					str,
				)
				if i != int(MuteDeafenCapture) && i != int(MuteDeafenWorker) {
					official += num
				}
			}
		} else {
			ch <- prometheus.MustNewConstMetric(
				c.counterDesc,
				prometheus.CounterValue,
				float64(official),
				str,
			)
		}
	}
}

func RecordDiscordRequests(client *redis.Client, requestType EventType, num int64) {
	for i := int64(0); i < num; i++ {
		typeStr := MetricTypeStrings[requestType]
		client.Incr(context.Background(), rediskey.RequestsByType(typeStr))
	}
}

func RecordDiscordRequest(client *redis.Client, requestType EventType) {
	typeStr := MetricTypeStrings[requestType]
	client.Incr(context.Background(), rediskey.RequestsByType(typeStr))
}

func NewCollector(client *redis.Client) *Collector {
	return &Collector{
		counterDesc: prometheus.NewDesc("discord_requests_by_type", "Number of discord requests made, differentiated by type", []string{"type"}, nil),
		client:      client,
	}
}

func PrometheusMetricsServer(client *redis.Client, port string) error {
	prometheus.MustRegister(NewCollector(client))

	http.Handle("/metrics", promhttp.Handler())

	return http.ListenAndServe(":"+port, nil)
}
