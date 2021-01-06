package galactus

import (
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/gorilla/mux"
	"github.com/jonas747/dshardmanager"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func SendChannelMessageHandler(logger *zap.Logger, shardManager *dshardmanager.Manager) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		channelID := vars["channelID"]
		valid, err := validate.ValidSnowflake(channelID)
		if !valid {
			errMsg := "channelID provided to sendMessageHandler is invalid"
			logger.Error(errMsg,
				zap.String("channelID", channelID),
				zap.Error(err),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errMsg := "could not read http body with error"
			logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		defer r.Body.Close()

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess, err := getRandomSession(shardManager)
		if err != nil {
			errMsg := "error obtaining random session for sendMessageHandler"
			logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		msg, err := sess.ChannelMessageSend(channelID, string(body))
		if err != nil {
			errMsg := "error posting message to channel"
			logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("contents", string(body)),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		logger.Info("posted message to channel",
			zap.String("channelID", channelID),
			zap.String("contents", string(body)),
			zap.String("messageID", msg.ID),
		)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(msg.ID))
	}
}
