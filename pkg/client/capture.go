package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/capture"
	"github.com/automuteus/utils/pkg/discord"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

func (galactus *GalactusClient) AddCaptureEvent(connectCode string, event capture.Event) error {
	str := strconv.FormatInt(int64(event.EventType), 10)
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.GetCaptureEventPartial, connectCode, str)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBuffer(event.Payload))
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return err
	}

	return err
}

func (galactus *GalactusClient) GetCaptureEvent(connectCode string) (*capture.Event, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.GetCaptureEventPartial, connectCode)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return nil, err
	}

	var event capture.Event
	err = json.Unmarshal(respBytes, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (galactus *GalactusClient) GetCaptureTask(connectCode string) (*discord.ModifyTask, error) {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.GetCaptureTaskPartial, connectCode)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return nil, err
	}

	var task discord.ModifyTask
	err = json.Unmarshal(respBytes, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (galactus *GalactusClient) SetCaptureTaskStatus(taskID, status string) error {
	url := endpoint.FormGalactusURL(galactus.Address, endpoint.CaptureRoute, endpoint.SetCaptureTaskStatusPartial, taskID)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBufferString(status))
	if err != nil {
		return err
	}
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		galactus.logger.Error("error reading all bytes from message body",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 response code received for " + url)
		return err
	}

	return err
}
