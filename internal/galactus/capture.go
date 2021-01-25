package galactus

import (
	"context"
	"errors"
	redisutils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/capture"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
	"time"
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
		galactus.logger.Info("added capture event",
			zap.String("connectCode", connectCode),
			zap.ByteString("event", body),
		)

		w.WriteHeader(http.StatusOK)
	}
}

func (galactus *GalactusAPI) GetCaptureEventHandler(timeout time.Duration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		connectCode := validate.ConnectCodeAndRespond(galactus.logger, w, r, endpoint.GetCaptureEventFull)
		if connectCode == "" {
			return
		}

		msg, err := capture.PopRawEvent(context.Background(), galactus.client, connectCode, timeout)

		// no jobs available
		switch {
		case errors.Is(err, redis.Nil):
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("{\"status\": \"No capture client events available\"}"))
			return
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
			galactus.logger.Error("redis error when popping capture event",
				zap.String("endpoint", endpoint.GetCaptureEventFull),
				zap.Error(err))
			return
		case msg == "":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"Nil capture task returned, despite no Redis errors\"}"))
			galactus.logger.Error("nil capture task returned, despite no Redis errors",
				zap.String("endpoint", endpoint.GetCaptureEventFull))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(msg))
		if err != nil {
			galactus.logger.Error("failed to write capture event as HTTP response",
				zap.String("endpoint", endpoint.GetCaptureEventFull),
				zap.Error(err),
			)
			return
		}
		galactus.logger.Info("popped capture event",
			zap.String("connectCode", connectCode),
			zap.String("event", msg),
		)
	}
}

func (galactus *GalactusAPI) GetCaptureTaskHandler(taskTimeout time.Duration) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		connectCode := validate.ConnectCodeAndRespond(galactus.logger, w, r, endpoint.GetCaptureTaskFull)
		if connectCode == "" {
			return
		}

		msg, err := redisutils.PopRawCaptureClientTask(galactus.client, connectCode, taskTimeout)

		// no jobs available
		switch {
		case errors.Is(err, redis.Nil):
			w.WriteHeader(http.StatusAccepted)
			w.Write([]byte("{\"status\": \"No capture client tasks available\"}"))
			return
		case err != nil:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"" + err.Error() + "\"}"))
			galactus.logger.Error("redis error when popping capture task",
				zap.String("endpoint", endpoint.GetCaptureTaskFull),
				zap.Error(err))
			return
		case msg == "":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{\"error\": \"Nil capture task returned, despite no Redis errors\"}"))
			galactus.logger.Error("nil capture task returned, despite no Redis errors",
				zap.String("endpoint", endpoint.GetCaptureTaskFull))
			return
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			galactus.logger.Error("failed to write capture task as HTTP response",
				zap.String("endpoint", endpoint.GetCaptureTaskFull),
				zap.Error(err),
			)
			return
		}
		galactus.logger.Info("popped capture task",
			zap.String("connectCode", connectCode),
			zap.String("task", msg),
		)
		w.WriteHeader(http.StatusOK)
	}
}

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
		galactus.logger.Info("wrote task status",
			zap.String("taskID", taskID),
			zap.String("value", out),
		)
		w.WriteHeader(http.StatusOK)
	}
}
