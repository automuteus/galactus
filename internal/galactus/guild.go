package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
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

func (galactus *GalactusAPI) GetGuildChannelsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildChannelsFull)
		if guildID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.GetGuildChannelsFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
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

func (galactus *GalactusAPI) GetGuildEmojisHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildEmojisFull)
		if guildID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.GetGuildEmojisFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}
		emojis, err := sess.GuildEmojis(guildID)
		if err != nil {
			errMsg := "failed to fetch guild emojis"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("guildID", guildID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		jBytes, err := json.Marshal(emojis)
		if err != nil {
			errMsg := "failed to marshal guild emojis to JSON"
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

func (galactus *GalactusAPI) GetGuildMemberHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID, userID := validate.GuildAndUserIDsAndRespond(galactus.logger, w, r, endpoint.GetGuildMemberFull)
		if guildID == "" || userID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.GetGuildMemberFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
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

func (galactus *GalactusAPI) GetGuildRolesHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		guildID := validate.GuildIDAndRespond(galactus.logger, w, r, endpoint.GetGuildRolesFull)
		if guildID == "" {
			return
		}

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.GetGuildRolesFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
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

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.CreateGuildEmojiFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
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
