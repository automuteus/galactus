package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) GetGuildChannelsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildChannelsFull)
		if guildID == "" {
			return
		}

		sess, err := getRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for getGuildChannels"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		channels, err := sess.GuildChannels(guildID)
		if err != nil {
			errMsg := "failed to fetch guild channels"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		jBytes, err := json.Marshal(channels)
		if err != nil {
			errMsg := "failed to marshal guild channels to JSON"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jBytes)
	}
}
