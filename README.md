# Docker socat Port Forward

Docker image for setting up one or multiple TCP ports forwarding, using socat.

## Getting started

- The ports mappings are set with environment variables, whose key must start with `PORT`, and then can have any name.
- Each environment variable can hold only one mapping. For setting multiple ports, many variables must be defined.
- The format of environment variable values is: `LOCAL_PORT:REMOTE_HOST:REMOTE_PORT` (LOCAL_PORT is optional, if not given, will use the same port as REMOTE_PORT)

### Example

Let's say you want to forward the following TCP ports:

- Remote port 9000 from remote host 192.168.0.10, to local (container) port 9999
- Remote port 8080 from remote host 192.168.0.100, lo local (container) port 8080

Then you can define these two environment variables, respectively (keys are examples and their values do not matter, as long as they start with "PORT"):

- `PORT1=9999:192.168.0:10:9000`
- `PORT_B=192.168.0.100:8080` (as we use the same local and remote port, local port can be undefined)

The complete Docker Run command would be the following:

```bash
docker run -d --name=portforward --net=host -e PORT1="9999:192.168.0:10:9000" -e PORT_B="192.168.0.100:8080" ghcr.io/david-lor/portforward
```

### Socks proxy support

The environment variable `SOCKS_PROXY` can be used for specifying the `ip:port` of a SOCKSv4 proxy to use for reaching the remote port.
This will be applied to ALL the port mappings on the current container.

## TODO

- Multiarch images

## Changelog

- 0.0.2
    - Fix socat command
- 0.0.1
    - Initial release
