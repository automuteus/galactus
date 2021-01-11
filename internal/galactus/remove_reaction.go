package galactus

import (
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) RemoveReactionHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.RemoveReactionFull)
		if channelID == "" || messageID == "" {
			return
		}
		// manually fetch the userID and emojiID, because they can be weird ("@me", or Unicode emoji)

		vars := mux.Vars(r)
		emojiID := vars["emojiID"]
		userID := vars["userID"]

		valid, err := validate.ValidSnowflake(userID)
		if !valid && userID != "@me" {
			errMsg := "userID is invalid and not @me for removeReaction"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		sess, err := getRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for removeReaction"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		err = sess.MessageReactionRemove(channelID, messageID, emojiID, userID)
		if err != nil {
			errMsg := "failed to removeReaction"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
				zap.String("emojiID", emojiID),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
