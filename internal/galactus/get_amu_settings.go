package galactus

import (
	"encoding/json"
	redis_utils "github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) GetGuildAMUSettings() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildAMUSettingsFull)
		if guildID == "" {
			return
		}

		sett, err := redis_utils.GetSettingsFromRedis(galactus.client, guildID)
		if err != nil {
			errMsg := "error when fetching guild AMU settings"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		jBytes, err := json.Marshal(sett)
		if err != nil {
			galactus.logger.Error("encountered an impossible error when marshalling guild settings that were just unmarshalled...",
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(jBytes)
		}
	}
}
