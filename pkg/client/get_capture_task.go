package galactus_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/utils/pkg/discord"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusClient) GetCaptureTask(connectCode string) (*discord.ModifyTask, error) {
	resp, err := galactus.client.Post(galactus.Address+endpoint.GetCaptureTaskPartial+connectCode, "application/json", bytes.NewBufferString(""))
	if err != nil {
		return nil, err
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading all bytes from resp body for getCaptureTask")
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := errors.New("non-200 status code received for GetCaptureTask:")
		return nil, err
	}

	var task discord.ModifyTask
	err = json.Unmarshal(respBytes, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}
