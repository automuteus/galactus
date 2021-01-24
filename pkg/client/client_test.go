package galactus_client

import (
	"github.com/automuteus/utils/pkg/capture"
	"go.uber.org/zap"
	"log"
	"sync"
	"testing"
	"time"
)

const TOTAL_TASKS = 10000

func TestNewGalactusClientRepeatedPolling(t *testing.T) {
	logger, _ := zap.NewProduction()

	client, err := NewGalactusClient("http://localhost:5858", logger)
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	// even if we never made any handlers, this shouldn't crash
	err = client.StartDiscordPolling()
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	err = client.StartDiscordPolling()
	if err == nil {
		log.Println("expected error thrown from starting polling after already polling")
		t.Fail()
	}

	client.StopDiscordPolling()
	client.StopAllPolling()
}

func TestNewGalactusClientCapture(t *testing.T) {
	logger, _ := zap.NewProduction()

	client, err := NewGalactusClient("http://localhost:5858", logger)
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	counter := 0

	wg := sync.WaitGroup{}
	wg.Add(1)

	f := func(event capture.Event) {
		counter++
		if counter == TOTAL_TASKS {
			wg.Done()
		}

	}
	cCode := "ABCDEFGH"

	client.RegisterCaptureHandler(cCode, f)

	start := time.Now()
	for i := 0; i < TOTAL_TASKS; i++ {
		err := client.AddCaptureEvent(cCode, capture.Event{
			EventType: capture.GameOver,
			Payload:   nil,
		})
		if err != nil {
			log.Println(err)
			t.Fail()
		}
	}
	log.Println(time.Now().Sub(start).String() + " to add all events")

	start = time.Now()
	err = client.StartCapturePolling(cCode)
	if err != nil {
		log.Println(err)
		t.Fail()
	}

	wg.Wait()
	log.Println(time.Now().Sub(start).String() + " to process all events")

	//client.StopAllPolling()
	client.StopCapturePolling(cCode)

}
