package galactus

import (
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) DeleteChannelMessageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.DeleteMessageFull)
		if channelID == "" || messageID == "" {
			return
		}

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for " + endpoint.DeleteMessageFull
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
