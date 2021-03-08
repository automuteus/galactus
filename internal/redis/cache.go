package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"time"
)

func IsEmbedEditUnique(client *redis.Client, channelID, messageID string, msg *discordgo.MessageEmbed) (bool, error) {
	oldHash := getEmbedHash(client, channelID, messageID)
	newHash := hash(msg)

	if newHash != oldHash {
		return true, writeEmbedHash(client, channelID, messageID, newHash)
	}
	return false, nil
}

func hash(msg *discordgo.MessageEmbed) string {
	if msg == nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(msg.Title))
	h.Write([]byte(msg.Description))
	for _, v := range msg.Fields {
		if v != nil {
			h.Write([]byte(v.Name))
			h.Write([]byte(v.Value))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TODO move to utils
func EmbedHashKey(channelID, messageID string) string {
	return "automuteus:hash:embed:" + channelID + ":" + messageID
}

func getEmbedHash(client *redis.Client, channelID, messageID string) string {
	// we actually don't care about the error here. Just assume no cache entry, embed is uncached
	r, _ := client.Get(context.Background(), EmbedHashKey(channelID, messageID)).Result()
	return r
}

func writeEmbedHash(client *redis.Client, channelID, messageID, hash string) error {
	return client.Set(context.Background(), EmbedHashKey(channelID, messageID), hash, time.Second*30).Err()
}
