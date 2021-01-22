package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/capture"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

type GalactusClient struct {
	Address string
	logger  *zap.Logger
	client  http.Client

	// we'll have capture channels for every relevant connect code
	captureKillChannels map[string]chan struct{}
	discordKillChannel  chan struct{}

	messageCreateHandlers      []func(m discordgo.MessageCreate)
	messageReactionAddHandlers []func(m discordgo.MessageReactionAdd)
	voiceStateUpdateHandlers   []func(m discordgo.VoiceStateUpdate)
	guildDeleteHandlers        []func(m discordgo.GuildDelete)
	guildCreateHandlers        []func(m discordgo.GuildCreate)

	genericCaptureHandlers map[string][]func(msg capture.Event)
}

func NewGalactusClient(address string, logger *zap.Logger) (*GalactusClient, error) {
	gc := GalactusClient{
		Address: address,
		logger:  logger,
		client:  http.Client{
			// Note: any relevant config here
		},
		captureKillChannels: make(map[string]chan struct{}),
		discordKillChannel:  nil,

		messageCreateHandlers:      make([]func(m discordgo.MessageCreate), 0),
		messageReactionAddHandlers: make([]func(m discordgo.MessageReactionAdd), 0),
		voiceStateUpdateHandlers:   make([]func(m discordgo.VoiceStateUpdate), 0),
		guildDeleteHandlers:        make([]func(m discordgo.GuildDelete), 0),
		guildCreateHandlers:        make([]func(m discordgo.GuildCreate), 0),
		genericCaptureHandlers:     make(map[string][]func(m capture.Event)),
	}
	r, err := http.Get(gc.Address + endpoint.GeneralRoute + "/")
	if err != nil {
		return &gc, err
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return &gc, errors.New("galactus returned a non-200 status code; ensure it is reachable")
	}
	return &gc, nil
}

type PollingType int

const (
	DiscordPolling PollingType = iota
	CapturePolling
)

func (galactus *GalactusClient) StartPolling(pollingType PollingType, connectCode string) error {
	var channel chan struct{}
	if pollingType == CapturePolling {
		valid, err := validate.ValidConnectCode(connectCode)
		if !valid {
			return err
		}
		if _, ok := galactus.captureKillChannels[connectCode]; ok {
			return errors.New("already polling for capture events for connect code " + connectCode)
		}
		galactus.captureKillChannels[connectCode] = make(chan struct{})
		channel = galactus.captureKillChannels[connectCode]
	} else {
		if galactus.discordKillChannel != nil {
			return errors.New("already polling for discord events")
		}
		galactus.discordKillChannel = make(chan struct{})
		channel = galactus.discordKillChannel
	}

	connected := false
	requestCancelChannel := make(chan struct{})

	go func() {
		for {
			select {
			case <-channel:
				requestCancelChannel <- struct{}{}
				return

			default:
				var url string
				switch pollingType {
				case DiscordPolling:
					url = endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.DiscordJobRequest)
				case CapturePolling:
					url = endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.GetCaptureEventPartial, connectCode)
				}
				req, err := http.NewRequest("POST", url, bytes.NewBufferString(""))
				if err != nil {
					galactus.logger.Error("invalid url provided to galactus client",
						zap.String("url", url))
					break
				}
				req.Cancel = requestCancelChannel

				response, err := http.DefaultClient.Do(req)
				if err != nil {
					connected = false
					galactus.logger.Error("could not reach galactus",
						zap.Error(err),
						zap.String("url", url))
					galactus.logger.Info("waiting 1 second before retrying")
					time.Sleep(time.Second * 1)
				} else {
					if !connected {
						galactus.logger.Info("successful connection to galactus")
						connected = true
					}
					body, err := ioutil.ReadAll(response.Body)
					if err != nil {
						galactus.logger.Error("error reading http response from galactus",
							zap.Error(err),
							zap.String("url", url),
							zap.ByteString("message", body))
					} else if response.StatusCode == http.StatusOK {
						switch pollingType {
						case DiscordPolling:
							var msg discord_message.DiscordMessage
							err := json.Unmarshal(body, &msg)
							if err != nil {
								galactus.logger.Error("error unmarshalling discord message from galactus",
									zap.Error(err),
									zap.ByteString("message", body))
							} else {
								galactus.dispatchDiscordMessage(msg)
							}
						case CapturePolling:
							var msg capture.Event
							err := json.Unmarshal(body, &msg)
							if err != nil {
								galactus.logger.Error("error unmarshalling capture message from galactus",
									zap.Error(err),
									zap.ByteString("message", body))
							} else {
								galactus.dispatchCaptureMessage(connectCode, msg)
							}
						}

					}
					response.Body.Close()
				}
			}
		}
	}()
	return nil
}

func (galactus *GalactusClient) dispatchDiscordMessage(msg discord_message.DiscordMessage) {
	switch msg.MessageType {
	case discord_message.MessageCreate:
		var messageCreate discordgo.MessageCreate
		err := json.Unmarshal(msg.Data, &messageCreate)
		if err != nil {
			galactus.logger.Error("error unmarshalling message data to MessageCreate",
				zap.Error(err),
				zap.ByteString("data", msg.Data))
		} else {
			for _, v := range galactus.messageCreateHandlers {
				v(messageCreate)
			}
		}
	case discord_message.MessageReactionAdd:
		var messageReactionAdd discordgo.MessageReactionAdd
		err := json.Unmarshal(msg.Data, &messageReactionAdd)
		if err != nil {
			galactus.logger.Error("error unmarshalling message data to MessageReactionAdd",
				zap.Error(err),
				zap.ByteString("data", msg.Data))
		} else {
			for _, v := range galactus.messageReactionAddHandlers {
				v(messageReactionAdd)
			}
		}
	case discord_message.VoiceStateUpdate:
		var voiceStateUpdate discordgo.VoiceStateUpdate
		err := json.Unmarshal(msg.Data, &voiceStateUpdate)
		if err != nil {
			galactus.logger.Error("error unmarshalling message data to VoiceStateUpdate",
				zap.Error(err),
				zap.ByteString("data", msg.Data))
		} else {
			for _, v := range galactus.voiceStateUpdateHandlers {
				v(voiceStateUpdate)
			}
		}
	case discord_message.GuildDelete:
		var guildDelete discordgo.GuildDelete
		err := json.Unmarshal(msg.Data, &guildDelete)
		if err != nil {
			galactus.logger.Error("error unmarshalling message data to GuildDelete",
				zap.Error(err),
				zap.ByteString("data", msg.Data))
		} else {
			for _, v := range galactus.guildDeleteHandlers {
				v(guildDelete)
			}
		}
	case discord_message.GuildCreate:
		var guildCreate discordgo.GuildCreate
		err := json.Unmarshal(msg.Data, &guildCreate)
		if err != nil {
			galactus.logger.Error("error unmarshalling message data to GuildCreate",
				zap.Error(err),
				zap.ByteString("data", msg.Data))
		} else {
			for _, v := range galactus.guildCreateHandlers {
				v(guildCreate)
			}
		}
	}
}

func (galactus *GalactusClient) dispatchCaptureMessage(connectCode string, msg capture.Event) {
	if handlers, ok := galactus.genericCaptureHandlers[connectCode]; ok {
		for _, v := range handlers {
			v(msg)
		}
	}
}

func (galactus *GalactusClient) StopCapturePolling(connectCode string) {
	if galactus.captureKillChannels[connectCode] != nil {
		galactus.captureKillChannels[connectCode] <- struct{}{}
	}
	delete(galactus.captureKillChannels, connectCode)
}

func (galactus *GalactusClient) StopDiscordPolling() {
	if galactus.discordKillChannel != nil {
		galactus.discordKillChannel <- struct{}{}
	}
	galactus.discordKillChannel = nil
}

func (galactus *GalactusClient) StopAllPolling() {
	galactus.StopDiscordPolling()
	for i := range galactus.captureKillChannels {
		galactus.StopCapturePolling(i)
	}
}

func (galactus *GalactusClient) RegisterDiscordHandler(msgType discord_message.DiscordMessageType, f interface{}) bool {
	registered := false
	switch msgType {
	case discord_message.MessageCreate:
		galactus.messageCreateHandlers = append(galactus.messageCreateHandlers, f.(func(m discordgo.MessageCreate)))
		registered = true
	case discord_message.MessageReactionAdd:
		galactus.messageReactionAddHandlers = append(galactus.messageReactionAddHandlers, f.(func(m discordgo.MessageReactionAdd)))
		registered = true
	case discord_message.GuildDelete:
		galactus.guildDeleteHandlers = append(galactus.guildDeleteHandlers, f.(func(m discordgo.GuildDelete)))
		registered = true
	case discord_message.VoiceStateUpdate:
		galactus.voiceStateUpdateHandlers = append(galactus.voiceStateUpdateHandlers, f.(func(m discordgo.VoiceStateUpdate)))
		registered = true
	case discord_message.GuildCreate:
		galactus.guildCreateHandlers = append(galactus.guildCreateHandlers, f.(func(m discordgo.GuildCreate)))
		registered = true
	}
	if registered {
		galactus.logger.Info("discord message handler registered",
			zap.String("type", discord_message.DiscordMessageTypeStrings[msgType]))
	} else {
		galactus.logger.Error("discord message handler type not recognized, handler not registered",
			zap.Int("type", int(msgType)))
	}
	return registered
}

func (galactus *GalactusClient) RegisterCaptureHandler(connectCode string, f interface{}) bool {
	if handlers, ok := galactus.genericCaptureHandlers[connectCode]; ok {
		handlers = append(handlers, f.(func(msg capture.Event)))
		galactus.genericCaptureHandlers[connectCode] = handlers
	} else {
		galactus.genericCaptureHandlers[connectCode] = make([]func(msg capture.Event), 1)
		galactus.genericCaptureHandlers[connectCode][0] = f.(func(msg capture.Event))
	}
	galactus.logger.Info("generic capture message handler registered")
	return true
}
