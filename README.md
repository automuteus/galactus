# galactus

> **This repository is deprecated.** Galactus now lives in the main
> [automuteus/automuteus](https://github.com/automuteus/automuteus) repository, under
> [`cmd/galactus`](https://github.com/automuteus/automuteus/tree/master/cmd/galactus) (entrypoint) and
> [`internal/broker`](https://github.com/automuteus/automuteus/tree/master/internal/broker) (the socket.io broker).
> It is built and released together with the bot, under the same version tag. No further changes will be made here.
>
> **The Docker image has not moved.** It is still published as
> [`automuteus/galactus`](https://hub.docker.com/r/automuteus/galactus) on Docker Hub; only the source has moved.
> Starting with the first release cut from the merged repository (8.5.0), its tags match the bot's
> (`automuteus/automuteus:X.Y.Z` and `automuteus/galactus:X.Y.Z` are built from the same commit).
>
> Please open issues and pull requests against [automuteus/automuteus](https://github.com/automuteus/automuteus).
> Self-hosting instructions are in [automuteus/deploy](https://github.com/automuteus/deploy).

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
