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

		sess, err := getRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for getGuildMember"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
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
		w.WriteHeader(http.StatusOK)
		w.Write(jBytes)
	}
}
