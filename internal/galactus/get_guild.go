package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

func (galactus *GalactusAPI) GetGuildHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildFull)
		if guildID == "" {
			return
		}

		id, err := strconv.ParseInt(guildID, 10, 64)
		if err != nil {
			errMsg := "failed to parse guildID as int64"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		sess := galactus.shardManager.SessionForGuild(id)
		guild, err := sess.State.Guild(guildID)
		if err != nil {
			errMsg := "failed to fetch guild from session state"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO fetch the guild with an actual API call here? if it fails via state?

		jBytes, err := json.Marshal(guild)
		if err != nil {
			errMsg := "failed to marshal guild to JSON"
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
