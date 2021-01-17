package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/capture"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusClient) GetCaptureEvent(connectCode string) (*capture.Event, error) {
	resp, err := galactus.client.Post(galactus.Address+endpoint.GetCaptureEventPartial+connectCode, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading all bytes from resp body for getCaptureEvent")
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 status code received for GetCaptureEvent:")
		return nil, err
	}

	var event capture.Event
	err = json.Unmarshal(respBytes, &event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}
