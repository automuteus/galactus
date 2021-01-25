package galactus

import (
	"encoding/json"
	"github.com/automuteus/galactus/internal/redis"
	"github.com/automuteus/galactus/pkg/endpoint"
	"github.com/automuteus/galactus/pkg/validate"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

func (galactus *GalactusAPI) SendChannelMessageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := validate.ChannelIDAndRespond(galactus.logger, w, r, endpoint.SendMessageFull)
		if channelID == "" {
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

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.SendMessageFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}

		RecordDiscordRequest(galactus.client, MessageCreate)
		msg, err := sess.ChannelMessageSend(channelID, string(body))
		if err != nil {
			errMsg := "error posting message to channel"
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
		galactus.logger.Info("posted message to channel",
			zap.String("channelID", channelID),
			zap.String("contents", string(body)),
			zap.String("messageID", msg.ID),
		)

		w.WriteHeader(http.StatusOK)
		jbytes, err := json.Marshal(msg)
		if err != nil {
			galactus.logger.Error("failed to marshal message to JSON",
				zap.Error(err),
			)
		}
		w.Write(jbytes)
	}
}

func (galactus *GalactusAPI) SendChannelMessageEmbedHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID := validate.ChannelIDAndRespond(galactus.logger, w, r, endpoint.SendMessageEmbedFull)
		if channelID == "" {
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

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.SendMessageEmbedFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}

		RecordDiscordRequest(galactus.client, MessageEmbedCreate)
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
		jbytes, err := json.Marshal(msg)
		if err != nil {
			galactus.logger.Error("failed to marshal embed message to JSON",
				zap.Error(err),
			)
		}
		w.Write(jbytes)
	}
}

func (galactus *GalactusAPI) EditMessageEmbedHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.EditMessageEmbedFull)
		if channelID == "" || messageID == "" {
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

		unique, err := redis.IsEmbedEditUnique(galactus.client, channelID, messageID, &embed)
		if err != nil {
			galactus.logger.Error("error when checking editEmbed uniqueness",
				zap.Error(err),
			)
		}
		if !unique {
			galactus.logger.Info("hash of message embed matched previous value - not editing message for the same contents",
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
			)
			w.WriteHeader(http.StatusAlreadyReported)
			return
		}

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.EditMessageEmbedFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}
		RecordDiscordRequest(galactus.client, MessageEmbedEdit)
		msg, err := sess.ChannelMessageEditEmbed(channelID, messageID, &embed)
		if err != nil {
			errMsg := "error editing message in channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		galactus.logger.Info("edited message in channel",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
		)
		w.WriteHeader(http.StatusOK)

		jbytes, err := json.Marshal(msg)
		if err != nil {
			galactus.logger.Error("failed to marshal edit embed message to JSON",
				zap.Error(err),
			)
		}
		w.Write(jbytes)
	}
}

func (galactus *GalactusAPI) DeleteChannelMessageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		channelID, messageID := validate.ChannelAndMessageIDsAndRespond(galactus.logger, w, r, endpoint.DeleteMessageFull)
		if channelID == "" || messageID == "" {
			return
		}

		// TODO perform some validation on the message body?
		// ex message length, empty contents, etc

		sess := galactus.shardManager.Session(0)
		if sess == nil {
			errMsg := "error obtaining session 0 for " + endpoint.DeleteMessageFull
			galactus.logger.Error(errMsg)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg))
			return
		}
		RecordDiscordRequest(galactus.client, MessageDelete)
		err := sess.ChannelMessageDelete(channelID, messageID)
		if err != nil {
			errMsg := "error deleting message in channel"
			galactus.logger.Error(errMsg,
				zap.Error(err),
				zap.String("channelID", channelID),
				zap.String("messageID", messageID),
			)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(errMsg + ": " + err.Error()))
			return
		}

		// TODO metrics logging here
		galactus.logger.Info("deleted message in channel",
			zap.String("channelID", channelID),
			zap.String("messageID", messageID),
		)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(messageID))
	}
}
