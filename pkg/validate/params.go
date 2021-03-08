package validate

import (
	"github.com/automuteus/utils/pkg/capture"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

func ChannelAndMessageIDsAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) (string, string) {
	vars := mux.Vars(r)
	channelID := vars["channelID"]
	messageID := vars["messageID"]
	valid, err := ValidSnowflake(channelID)
	if !valid {
		errMsg := "channelID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("channelID", channelID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return "", ""
	}
	valid, err = ValidSnowflake(messageID)
	if !valid {
		errMsg := "messageID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("messageID", messageID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return channelID, ""
	}
	return channelID, messageID
}

func GuildAndUserIDsAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) (string, string) {
	vars := mux.Vars(r)
	guildID := vars["guildID"]
	userID := vars["userID"]
	valid, err := ValidSnowflake(guildID)
	if !valid {
		errMsg := "channelID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("channelID", guildID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return "", ""
	}
	valid, err = ValidSnowflake(userID)
	if !valid {
		errMsg := "userID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("userID", userID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return guildID, ""
	}
	return guildID, userID
}

func ChannelIDAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	channelID := vars["channelID"]
	valid, err := ValidSnowflake(channelID)
	if !valid {
		errMsg := "channelID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("channelID", channelID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return ""
	}
	return channelID
}

func GuildIDAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	guildID := vars["guildID"]
	valid, err := ValidSnowflake(guildID)
	if !valid {
		errMsg := "channelID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("channelID", guildID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return ""
	}
	return guildID
}

func UserIDAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	userID := vars["userID"]
	valid, err := ValidSnowflake(userID)
	if !valid {
		errMsg := "userID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.String("userID", userID),
			zap.String("endpoint", endpoint),
			zap.Error(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg + ": " + err.Error()))
		return ""
	}
	return userID
}

func NameAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	name := vars["name"]
	valid := name != ""
	if !valid {
		errMsg := "name provided to " + endpoint + " is empty and therefore invalid"
		logger.Error(errMsg,
			zap.String("name", name),
			zap.String("endpoint", endpoint),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return ""
	}
	return name
}

func ConnectCodeAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	connectCode := vars["connectCode"]
	valid, err := ValidConnectCode(connectCode)
	if !valid {
		errMsg := "connectCode provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.Error(err),
			zap.String("connectCode", connectCode),
			zap.String("endpoint", endpoint),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return ""
	}
	return connectCode
}

func TaskIDAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) string {
	vars := mux.Vars(r)
	taskID := vars["taskID"]
	valid, err := ValidTaskID(taskID)
	if !valid {
		errMsg := "taskID provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.Error(err),
			zap.String("taskID", taskID),
			zap.String("endpoint", endpoint),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return ""
	}
	return taskID
}

func EventTypeAndRespond(logger *zap.Logger, w http.ResponseWriter, r *http.Request, endpoint string) (bool, capture.EventType) {
	vars := mux.Vars(r)
	eventType := vars["eventType"]
	e, err := strconv.ParseInt(eventType, 10, 64)
	if err != nil {
		return false, 0
	}
	valid, err := ValidEventType(int(e))
	if !valid {
		errMsg := "eventType provided to " + endpoint + " is invalid"
		logger.Error(errMsg,
			zap.Error(err),
			zap.String("eventType", eventType),
			zap.String("endpoint", endpoint),
		)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return false, 0
	}
	return true, capture.EventType(e)
}
