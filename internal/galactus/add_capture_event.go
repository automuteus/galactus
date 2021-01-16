package galactus

import (
	"context"
	"encoding/json"
	"github.com/automuteus/galactus/pkg/capture_message"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/task"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusAPI) AddCaptureEventHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		connectCode := validate.ConnectCodeAndRespond(galactus.logger, w, r, endpoint.AddCaptureEventFull)
		if connectCode == "" {
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errMsg := "could not read http body with error"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		defer r.Body.Close()

		var message capture_message.CaptureMessage
		err = json.Unmarshal(body, &message)
		if err != nil {
			errMsg := "error unmarshalling CaptureMessage from JSON"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("body", string(body)),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		err = task.PushJob(context.Background(), galactus.client, connectCode, message.MessageType, string(message.Data))
		if err != nil {
			errMsg := "error pushing capture job to Redis"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("body", string(body)),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
