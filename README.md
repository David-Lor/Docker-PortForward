# Docker socat Port Forward

Docker image for setting up one or multiple TCP ports forwarding, using socat.

## Getting started

- The ports mappings are set with environment variables, whose key must start with `PORT`, and then can have any name.
- Each environment variable can hold only one mapping. For setting multiple ports, many variables must be defined.
- The format of environment variable values is: `LOCAL_PORT:REMOTE_HOST:REMOTE_PORT` (LOCAL_PORT is optional, if not given, will use the same port as REMOTE_PORT)
- If you're using a fork of this repo, you can build and pull your own images

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

### Port range

Multiple ports, on a given range in series, can be forwarded using a single environment variable. For doing so, using the same syntax as with normal ports, give a range with the format `START-END` (being START and END both included), in the place of the port.

For example, if you want to forward ports 1000 to 1010 from 192.168.0.10 to the same local ports (1000~1010 respectively),
you can define an environment variable like: `PORTS1=192.168.0.10:1000-1010`.

A range can also be specified for the local ports. In this case, the ranges of local and remote ports must have the same length.
For example, if you want to forward ports 1000 to 1010 from 192.168.0.10 to local ports 2000 to 2010 respectively,
you can define an environment variable like: `PORTS2=2000-2010:192.168.0.10:1000-1010`

### Socks proxy support

The environment variable `SOCKS_PROXY` can be used for specifying the `ip:port` of a SOCKSv4 proxy to use for reaching the remote port.
This will be applied to ALL the port mappings on the current container.

## TODO
- [X] Multiarch images

## Changelog

- 0.1.1
  - Port range forwarding (one socat command per port)
  - SOCKS proxy support (socat)
  - Add tests, integrated in GitHub Actions
- 0.0.2
  - Fix socat command
- 0.0.1
  - Initial release
