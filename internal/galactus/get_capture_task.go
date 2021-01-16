package galactus

import (
	"errors"
	redisutils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"net/http"
	"time"
)

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

		w.WriteHeader(http.StatusOK)

		_, err = w.Write([]byte(msg))
		if err != nil {
			galactus.logger.Error("failed to write capture task as HTTP response",
				zap.String("endpoint", endpoint.GetCaptureTaskFull),
				zap.Error(err),
			)
		}
	}
}
