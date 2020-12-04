# galactus
The All-Knowing Bot Token and Socket Provider Aggregator.

https://youtu.be/y8OnoxKotPQ

## Description

This project is comprised of two key servers for enabling crucial functionality of AutoMuteUs.

The "broker" is responsible for handling socket.io connections with capture clients. This broker handles all sockets, and
transmits relevant information to automuteus via Redis. This allows a complete decoupling of core bot functionality from sockets;
upgrades to the bot functionality can be performed without severing connections to capture clients.

The "galactus" server is responsible for issuing mute/deafen requests to Discord. In some sense, galactus acts as a
muting/deafening proxy; the automuteus bot fundamentally should not care how mutes or deafens are issued, just that they go through. Galactus
uses whatever methods available (capture-side bots, secondary bot tokens, or the primary bot) to accomplish these mute and deafen
requests.

## Environment Variables

### Required:
* `DISCORD_BOT_TOKEN`: The primary Bot Token to be used for mute/deafen requests if no other methods are applicable.
**This is the same bot token as used for AutoMuteUs!**
* `REDIS_ADDR`: The location at which Redis is reachable. Redis is used for a variety of purposes within Galactus, including
storage of temporary tokens, and, crucially, communication between the Capture connection broker and AutoMuteUs itself.

### Optional:
* `GALACTUS_PORT`: The port on which Galactus will run and receive requests from AutoMuteUs. Defaults to 5858.
* `BROKER_PORT`: The port on which the broker will listen for socket connections from capture clients. Defaults to 8123.
* `REDIS_USER`: Username to authenticate with Redis, if applicable.
* `REDIS_PASS`: Password to authenticate with Redis, if applicable.

## **Do not provide unless you know what you're doing**:
* `NUM_SHARDS`: Should match whatever automuteus is using
* `SHARD_ID`: Probably just use 0
* `MAX_REQ_5_SEC`: How many Discord API mute/deafens should be issued per token per 5 second window. Defaults to 7 (ratelimits
returned by Discord are anywhere from [5-10]/5sec, so 7 is a decent heuristic)
* `ACK_TIMEOUT_MS`: How many milliseconds after a Mute task is received before it times out, if no capture bot picks up the task
* `TASK_TIMEOUT_MS`: After a mute task is issued to a capture bot, how long before it times out and another method is used
