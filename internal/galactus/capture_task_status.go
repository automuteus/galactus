package galactus

import (
	"context"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/rediskey"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusAPI) SetCaptureTaskStatusHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := validate.TaskIDAndRespond(galactus.logger, w, r, endpoint.SetCaptureTaskStatusFull)
		if taskID == "" {
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		defer r.Body.Close()
		bodyStr := string(body)
		var out string

		if bodyStr == "true" || bodyStr == "t" {
			out = "true"
		} else {
			out = "false"
		}
		err = galactus.client.Publish(context.Background(), rediskey.CompleteTask(taskID), out).Err()
		if err != nil {
			errMsg := "failed to publish task status to Redis"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("taskID", taskID),
				zap.String("value", out),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
