package galactus

import (
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) RemoveAllReactionsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.RemoveAllReactionsFull)
		if channelID == "" || messageID == "" {
			return
		}

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for removeAllReactions"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		err = sess.MessageReactionsRemoveAll(channelID, messageID)
		if err != nil {
			errMsg := "failed to remove all reactions"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
