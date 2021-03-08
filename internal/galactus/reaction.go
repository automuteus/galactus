package galactus

import (
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

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.AddReactionFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}
		RecordDiscordRequest(galactus.client, ReactionAdd)
		err := sess.MessageReactionAdd(channelID, messageID, emojiID)
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
		galactus.logger.Info("added reaction to channel message",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
			zap.String("emojiID", emojiID),
		)

		w.WriteHeader(http.StatusOK)
	}
}

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

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.RemoveReactionFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}

		RecordDiscordRequest(galactus.client, ReactionRemove)
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
		galactus.logger.Info("removed reaction on channel message",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
			zap.String("emojiID", emojiID),
			zap.String("userID", userID),
		)

		w.WriteHeader(http.StatusOK)
	}
}

func (galactus *GalactusAPI) RemoveAllReactionsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.RemoveAllReactionsFull)
		if channelID == "" || messageID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.RemoveAllReactionsFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}

		RecordDiscordRequest(galactus.client, ReactionRemoveAll)
		err := sess.MessageReactionsRemoveAll(channelID, messageID)
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

		galactus.logger.Info("removed all reactions on channel message",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
		)

		w.WriteHeader(http.StatusOK)
	}
}
