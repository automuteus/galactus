package galactus

import (
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) DeleteChannelMessageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		channelID := vars["channelID"]
		messageID := vars["messageID"]
		valid, err := validate.ValidSnowflake(channelID)
		if !valid {
			errMsg := "channelID provided to deleteMessageHandler is invalid"
			galactus.logger.Error(errMsg,
				zap.String("channelID", channelID),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		valid, err = validate.ValidSnowflake(messageID)
		if !valid {
			errMsg := "messageID provided to deleteMessageHandler is invalid"
			galactus.logger.Error(errMsg,
				zap.String("messageID", messageID),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess, err := getRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for sendMessageHandler"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		err = sess.ChannelMessageDelete(channelID, messageID)
		if err != nil {
			errMsg := "error deleting message in channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		galactus.logger.Info("deleted message in channel",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
		)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(messageID))
	}
}
