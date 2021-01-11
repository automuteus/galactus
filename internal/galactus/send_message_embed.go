package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusAPI) SendChannelMessageEmbedHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		channelID := vars["channelID"]
		valid, err := validate.ValidSnowflake(channelID)
		if !valid {
			errMsg := "channelID provided to sendMessageEmbedHandler is invalid"
			galactus.logger.Error(errMsg,
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
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		defer r.Body.Close()

		var embed discordgo.MessageEmbed
		err = json.Unmarshal(body, &embed)
		if err != nil {
			errMsg := "error unmarshalling discordMessageEmbed from JSON"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("body", string(body)),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO extra validation here (empty embed fields and the like)

		sess, err := getRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for sendMessageEmbedHandler"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		msg, err := sess.ChannelMessageSendEmbed(channelID, &embed)
		if err != nil {
			errMsg := "error posting messageEmbed to channel"
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
		galactus.logger.Info("posted messageEmbed to channel",
			zap.String("channelID", channelID),
			zap.String("contents", string(body)),
			zap.String("messageID", msg.ID),
		)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(msg.ID))
	}
}
