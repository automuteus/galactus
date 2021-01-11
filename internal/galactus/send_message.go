package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusAPI) SendChannelMessageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := validate.ChannelIDAndRespond(galactus.logger, w, r, endpoint.SendMessageFull)
		if channelID == "" {
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

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for sendMessageHandler"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		msg, err := sess.ChannelMessageSend(channelID, string(body))
		if err != nil {
			errMsg := "error posting message to channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("contents", string(body)),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		galactus.logger.Info("posted message to channel",
			zap.String("channelID", channelID),
			zap.String("contents", string(body)),
			zap.String("messageID", msg.ID),
		)
		w.WriteHeader(http.StatusOK)
		jbytes, err := json.Marshal(msg)
		if err != nil {
			log.Println(err)
		}
		w.Write(jbytes)
	}
}
