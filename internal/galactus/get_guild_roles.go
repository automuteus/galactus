package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) GetGuildRolesHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildRolesFull)
		if guildID == "" {
			return
		}

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for getGuildRoles"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		roles, err := sess.GuildRoles(guildID)
		if err != nil {
			errMsg := "failed to fetch guild roles"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		jBytes, err := json.Marshal(roles)
		if err != nil {
			errMsg := "failed to marshal guild roles to JSON"
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
