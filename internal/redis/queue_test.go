package redis

import (
	"errors"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"log"
	"strings"
	"testing"
)

const inputMsg = "{\"id\":\"0\"," +
	"\"channel_id\":\"1\"," +
	"\"guild_id\":\"2\"," +
	"\"content\":\"test\"," +
	"\"timestamp\":\"2020-12-30T22:37:43.404000+00:00\"," +
	"\"edited_timestamp\":\"\"," +
	"\"mention_roles\":[]," +
	"\"tts\":false," +
	"\"mention_everyone\":false," +
	"\"author\":{\"id\":\"3\",\"email\":\"\",\"username\":\"Soup\",\"avatar\":\"omitted\",\"locale\":\"\",\"discriminator\":\"1234\",\"token\":\"\",\"verified\":false,\"mfa_enabled\":false,\"bot\":false,\"public_flags\":0,\"premium_type\":0,\"system\":false,\"flags\":0}," +
	"\"attachments\":[]," +
	"\"embeds\":[]," +
	"\"mentions\":[]," +
	"\"reactions\":null," +
	"\"pinned\":false," +
	"\"type\":0," +
	"\"webhook_id\":\"\"," +
	"\"member\":{\"guild_id\":\"\",\"joined_at\":\"2016-01-25T07:32:22.570000+00:00\",\"nick\":\"\",\"deaf\":false,\"mute\":false,\"user\":null,\"roles\":[],\"premium_since\":\"\"}," +
	"\"mention_channels\":null," +
	"\"activity\":null," +
	"\"application\":null," +
	"\"message_reference\":null," +
	"\"flags\":0}"

func TestPopEmpty(t *testing.T) {
	client := newTestRedis()
	msg, err := PopDiscordMessage(client)

	if msg != nil {
		t.Fatal("non-nil message received from empty pop")
	}

	if err == nil {
		t.Fatal("nil error returned from empty pop")
	}

	if !errors.Is(err, redis.Nil) {
		t.Fatal("error returned from empty pop is not redis.Nil")
	}
}

func TestPushAndPopSingle(t *testing.T) {
	client := newTestRedis()

	err := PushDiscordMessage(client, MessageCreate, []byte(inputMsg))
	if err != nil {
		t.Fatal(err)
	}

	msg, err := PopDiscordMessage(client)
	if err != nil {
		t.Fatal(err)
	} else if msg == nil {
		t.Fatal("nil message returned when expected the previous msg we pushed")
	}

	if msg.MessageType != MessageCreate {
		t.Fatal("returned msg type is not msgcreate")
	}

	if !strings.EqualFold(inputMsg, string(msg.Data)) {
		t.Fatal("input and output messages are not equivalent")
	}
}

func TestPushAndPopMultiple(t *testing.T) {
	client := newTestRedis()

	err := PushDiscordMessage(client, MessageCreate, []byte(inputMsg))
	if err != nil {
		log.Fatal(err)
	}
	input2 := strings.Replace(inputMsg, "\"id\":\"0\"", "\"id\":\"1\"", 1)
	err = PushDiscordMessage(client, MessageCreate, []byte(input2))
	if err != nil {
		log.Fatal(err)
	}

	msg, err := PopDiscordMessage(client)
	if err != nil {
		log.Fatal(err)
	} else if msg == nil {
		log.Fatal("nil message returned when expected the previous msg we pushed")
	}

	if msg.MessageType != MessageCreate {
		t.Fatal("returned msg type is not msgcreate")
	}

	if !strings.EqualFold(inputMsg, string(msg.Data)) {
		t.Fatal("input and output messages are not equivalent")
	}

	msg, err = PopDiscordMessage(client)
	if err != nil {
		log.Fatal(err)
	} else if msg == nil {
		log.Fatal("nil message returned when expected the previous msg we pushed for input2")
	}

	if msg.MessageType != MessageCreate {
		t.Fatal("returned msg type is not msgcreate for input2")
	}

	if !strings.EqualFold(input2, string(msg.Data)) {
		t.Fatal("input2 and output messages are not equivalent")
	}

	// replace back; now the string comparison should fail
	input2 = strings.Replace(input2, "\"id\":\"1\"", "\"id\":\"0\"", 1)

	if strings.EqualFold(input2, string(msg.Data)) {
		t.Fatal("input and output messages are equivalent, when we mutated the input on purpose")
	}
}

// newTestRedis returns a redis.Cmdable.
func newTestRedis() *redis.Client {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return client
}
