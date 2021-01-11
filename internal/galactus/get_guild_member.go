package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"net/http"
)

func (galactus *GalactusAPI) GetGuildMemberHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID, userID := validate.GuildAndUserIDsAndRespond(galactus.logger, w, r, endpoint.GetGuildMemberFull)
		if guildID == "" || userID == "" {
			return
		}

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for getGuildMember"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		member, err := sess.GuildMember(guildID, userID)
		if err != nil {
			errMsg := "failed to fetch guild member"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		jBytes, err := json.Marshal(member)
		if err != nil {
			errMsg := "failed to marshal guild member"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
				zap.String("userID", userID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(jBytes)
	}
}
