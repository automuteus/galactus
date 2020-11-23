package discord

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"time"
)

type ModifyTask struct {
	GuildID    int64             `json:"guildID"`
	UserID     int64             `json:"userID"`
	Parameters NoNickPatchParams `json:"parameters"`
	TaskID     string            `json:"taskID"`
}

func NewModifyTask(guildID, userID int64, params NoNickPatchParams) ModifyTask {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", guildID)))
	h.Write([]byte(fmt.Sprintf("%d", userID)))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().Unix())))
	return ModifyTask{
		GuildID:    guildID,
		UserID:     userID,
		Parameters: params,
		TaskID:     hex.EncodeToString(h.Sum(nil))[0:10],
	}
}

type NoNickPatchParams struct {
	Deaf bool `json:"deaf"`
	Mute bool `json:"mute"`
}

func ApplyMuteDeaf(sess *discordgo.Session, guildID, userID string, mute, deaf bool) error {
	log.Printf("Issuing update request to discord for UserID %s with mute=%v deaf=%v\n", userID, mute, deaf)

	p := NoNickPatchParams{
		Deaf: deaf,
		Mute: mute,
	}

	_, err := sess.RequestWithBucketID("PATCH", discordgo.EndpointGuildMember(guildID, userID), p, discordgo.EndpointGuildMember(guildID, ""))
	return err
}
