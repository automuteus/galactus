package galactus

import (
	"context"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/capture"
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

		valid, eventType := validate.EventTypeAndRespond(galactus.logger, w, r, endpoint.AddCaptureEventFull)
		if !valid {
			errMsg := "invalid eventType provided"
			galactus.logger.Error(errMsg,
				zap.Int("eventType", int(eventType)),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg))
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

		// TODO more validation on the payload here?

		err = capture.PushEvent(context.Background(), galactus.client, connectCode, eventType, string(body))
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
