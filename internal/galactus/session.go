package galactus

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dshardmanager"
	"math/rand"
)

const MaxInvalidRandomSessions = 5

func getRandomSession(manager *dshardmanager.Manager) (*discordgo.Session, error) {
	max := manager.GetNumShards()
	sess := manager.Session(rand.Intn(max))
	i := 1

	for sess == nil {
		if i > MaxInvalidRandomSessions {
			return nil, errors.New("exceeded maximum retries for random session")
		}
		i++
		r := rand.Intn(max)
		sess = manager.Session(r)
	}
	return sess, nil
}
