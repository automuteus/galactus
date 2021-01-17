package galactus_client

import (
	"bytes"
	"fmt"
	"github.com/automuteus/galactus/pkg/capture_message"
	"github.com/automuteus/galactus/pkg/endpoint"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusClient) AddCaptureEvent(connectCode string, event capture_message.CaptureMessage) error {
	url := fmt.Sprintf("%s%s/%d", galactus.Address+endpoint.AddCaptureEventPartial, connectCode, event.MessageType)
	resp, err := galactus.client.Post(url, "application/json", bytes.NewBuffer(event.Data))
	if err != nil {
		return err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading all bytes from resp body for addcaptureevent")
		log.Println(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("non-200 status code received for addcaptureevent:")
		log.Println(string(respBytes))
	}

	return err
}
