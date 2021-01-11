package galactus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/automuteus/utils/pkg/rediskey"
	"github.com/automuteus/utils/pkg/settings"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) GetGuildAMUSettings() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildAMUSettingsFull)
		if guildID == "" {
			return
		}

		key := rediskey.GuildSettings(HashGuildID(guildID))
		var sett settings.GuildSettings

		str, err := galactus.client.Get(context.Background(), key).Result()
		if err != nil {
			errMsg := "error when fetching guild AMU settings"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		err = json.Unmarshal([]byte(str), &sett)
		if err != nil {
			errMsg := "error when unmarshalling guild AMU settings"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
				zap.String("data", str),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(str))
	}
}

func HashGuildID(guildID string) string {
	return genericHash(guildID)
}

func genericHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
