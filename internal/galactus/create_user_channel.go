package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) CreateUserChannelHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := validate.UserIDAndRespond(galactus.logger, w, r, endpoint.UserChannelCreateFull)
		if userID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.UserChannelCreateFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}
		channel, err := sess.UserChannelCreate(userID)
		if err != nil {
			errMsg := "failed to create user channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		jBytes, err := json.Marshal(channel)
		if err != nil {
			errMsg := "failed to marshal user channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		galactus.logger.Info("created user channel",
			zap.String("userID", userID),
			zap.String("channelID", channel.ID),
		)
		w.WriteHeader(http.StatusOK)
		w.Write(jBytes)
	}
}
