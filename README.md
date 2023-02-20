# galactus
The All-Knowing AutoMuteUs Socket Connection Aggregator.

https://youtu.be/y8OnoxKotPQ

## Description

Galactus is responsible for handling socket.io connections with capture clients. This broker handles all sockets, and
transmits relevant information to automuteus via Redis. This allows a complete decoupling of core bot functionality from sockets;
upgrades to the bot functionality can be performed without severing connections to capture clients.

## Environment Variables

### Required:
* `REDIS_ADDR`: The location at which Redis is reachable. Redis is used to communicate capture messages to the bot instance.

### Optional:
* `BROKER_PORT`: The port on which the broker will listen for socket connections from capture clients. Defaults to 8123.
* `REDIS_USER`: Username to authenticate with Redis, if applicable.
* `REDIS_PASS`: Password to authenticate with Redis, if applicable.