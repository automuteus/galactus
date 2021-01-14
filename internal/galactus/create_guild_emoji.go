package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/internal/galactus/shard_manager"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
)

func (galactus *GalactusAPI) CreateGuildEmojiHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.CreateGuildEmojiFull)
		name := validate.NameAndRespond(galactus.logger, w, r, endpoint.CreateGuildEmojiFull)
		if guildID == "" || name == "" {
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

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		// Addl. constraint for emojis: must be under 256kB

		sess, err := shard_manager.GetRandomSession(galactus.shardManager)
		if err != nil {
			errMsg := "error obtaining random session for sendMessageHandler"
			galactus.logger.Error(errMsg,
				zap.Error(err),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		emoji, err := sess.GuildEmojiCreate(guildID, name, string(body), nil)
		if err != nil {
			errMsg := "error creating emoji for guild"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
				zap.String("name", name),
				zap.String("emoji", string(body)),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		galactus.logger.Info("created emoji for guild",
			zap.String("guildID", guildID),
			zap.String("name", name),
			zap.String("emoji", string(body)),
			zap.String("emojiID", emoji.ID),
		)
		w.WriteHeader(http.StatusOK)
		jbytes, err := json.Marshal(emoji)
		if err != nil {
			log.Println(err)
		}
		w.Write(jbytes)
	}
}
