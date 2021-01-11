package galactus

import (
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) AddReactionHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.AddReactionFull)
		if channelID == "" || messageID == "" {
			return
		}

		// manually fetch the emojiID, because it can be a non-numeric/snowflake Unicode emoji

		vars := mux.Vars(r)
		emojiID := vars["emojiID"]

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for addReaction"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		err = sess.MessageReactionAdd(channelID, messageID, emojiID)
		if err != nil {
			errMsg := "failed to addReaction"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
				zap.String("emojiID", emojiID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
