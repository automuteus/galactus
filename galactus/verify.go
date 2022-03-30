package galactus

import (
	"context"
	"github.com/automuteus/utils/pkg/rediskey"
	"log"
	"time"
)

type botVerifyTask struct {
	guildID string
	limit   int
}

func (tokenProvider *TokenProvider) enqueueBotMembershipVerifyTask(guildID string, limit int) {
	tokenProvider.botVerificationQueue <- botVerifyTask{
		guildID: guildID,
		limit:   limit,
	}
}

func (tokenProvider *TokenProvider) verifyBotMembership(guildID string, limit int, uniqueTokensUsed map[string]struct{}) {
	tokenProvider.sessionLock.RLock()
	defer tokenProvider.sessionLock.RUnlock()

	i := 0
	for hToken, sess := range tokenProvider.activeSessions {
		// only check tokens that weren't used successfully already (obv we're members if mute/deafen was successful earlier)
		if !mapHasEntry(uniqueTokensUsed, hToken) {
			_, err := sess.GuildMember(guildID, sess.State.User.ID)
			if err != nil {
				//log.Println(err)
			} else {
				i++ // successfully checked self's membership; we are a member of this server
			}

			// if the bot is verified as a member of too many servers for the premium status, then we should leave them
			if i > limit {
				log.Println("Token/Bot " + hToken + " leaving server " + guildID + " due to lack of premium membership")

				err = sess.GuildLeave(guildID)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func (tokenProvider *TokenProvider) canRunBotVerification(guildID string) bool {
	v, err := tokenProvider.client.Exists(context.Background(), rediskey.GuildPremiumMembershipVerify(guildID)).Result()
	if err != nil {
		log.Println(err)
		return true
	}
	return v != 1 // 1 = exists, therefore don't run
}

func (tokenProvider *TokenProvider) markBotVerificationLockout(guildID string) error {
	// we only need to check the
	return tokenProvider.client.Set(context.Background(), rediskey.GuildPremiumMembershipVerify(guildID), 1, time.Hour*24).Err()
}
