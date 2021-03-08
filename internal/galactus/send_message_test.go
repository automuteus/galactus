package galactus

import (
	"fmt"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGalactusAPI_SendChannelMessageHandler(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Println("Failed to initialize logger with error")
		t.Fatal(err)
	}
	galactus := NewGalactusAPI(logger, true, os.Getenv("TEST_BOT_TOKEN"), "", "", "", 7)
	defer galactus.Close()

	m := mux.NewRouter()
	m.HandleFunc(endpoint.SendMessageFull, galactus.SendChannelMessageHandler())

	ts := httptest.NewServer(m)
	defer ts.Close()

	r := strings.NewReader("test message")

	resp, err := http.Post(ts.URL+endpoint.SendMessagePartial+os.Getenv("TEST_CHANNEL_ID"), "", r)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Fatal(fmt.Sprintf("non 200 status response: %d", resp.StatusCode))
	}

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
}
