package galactus_client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/discord_message"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/capture"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"time"
)

func (galactus *GalactusClient) StartCapturePolling(connectCode string) error {
	valid, err := validate.ValidConnectCode(connectCode)
	if !valid {
		return err
	}
	if _, ok := galactus.captureKillChannels[connectCode]; ok {
		return errors.New("already polling for capture events for connect code " + connectCode)
	}
	galactus.captureKillChannels[connectCode] = make(chan struct{})

	connected := false

	ctx, cancelRequest := context.WithCancel(context.Background())

	url := endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.GetCaptureEventPartial, connectCode)
	go func() {
		for {
			<-galactus.captureKillChannels[connectCode]
			cancelRequest()
			delete(galactus.captureKillChannels, connectCode)
			return
		}
	}()

	go func() {
		for {
			req, err := http.NewRequest("POST", url, bytes.NewBufferString(""))
			if err != nil {
				galactus.logger.Error("invalid url provided to galactus client",
					zap.String("url", url))
				break
			}
			// if we're told to stop polling, we'd better do so
			req.WithContext(ctx)

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
				response.Body.Close()
			}
		}
	}()
	return nil
}

func (galactus *GalactusClient) StartDiscordPolling() error {
	if galactus.discordKillChannel != nil {
		return errors.New("already polling for discord events")
	}
	galactus.discordKillChannel = make(chan struct{})

	connected := false
	ctx, cancelRequest := context.WithCancel(context.Background())

	url := endpoint.FormGalactusURL(galactus.Address, endpoint.DiscordRoute, endpoint.DiscordJobRequest)

	go func() {
		for {
			<-galactus.discordKillChannel
			cancelRequest()
			galactus.discordKillChannel = nil
			return
		}
	}()

	go func() {
		for {
			req, err := http.NewRequest("POST", url, bytes.NewBufferString(""))
			if err != nil {
				galactus.logger.Error("invalid url provided to galactus client",
					zap.String("url", url))
				break
			}
			req.WithContext(ctx)

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
					var msg discord_message.DiscordMessage
					err := json.Unmarshal(body, &msg)
					if err != nil {
						galactus.logger.Error("error unmarshalling discord message from galactus",
							zap.Error(err),
							zap.ByteString("message", body))
					} else {
						galactus.dispatchDiscordMessage(msg)
					}
				}
				response.Body.Close()
			}
		}
	}()
	return nil
}

func (galactus *GalactusClient) StopCapturePolling(connectCode string) {
	if galactus.captureKillChannels[connectCode] != nil {
		galactus.captureKillChannels[connectCode] <- struct{}{}
	}
}

func (galactus *GalactusClient) StopDiscordPolling() {
	if galactus.discordKillChannel != nil {
		galactus.discordKillChannel <- struct{}{}
	}
}

func (galactus *GalactusClient) StopAllPolling() {
	galactus.StopDiscordPolling()
	for i := range galactus.captureKillChannels {
		galactus.StopCapturePolling(i)
	}
}
