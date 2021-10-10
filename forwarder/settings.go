package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env var format: PORT=localport:remotehost:remoteport
const ENV_PREFIX = "PORT"

type PortForward struct {
	LocalPort  int64
	RemoteHost string
	RemotePort int64
}

func (p *PortForward) ToString() string {
	return fmt.Sprintf("%d:%s:%d", p.LocalPort, p.RemoteHost, p.RemotePort)
}

func getAllEnvironmentVariables() map[string]string {
	environment := make(map[string]string)
	for _, e := range os.Environ() {
		i := strings.Index(e, "=")
		if i >= 0 {
			environment[e[:i]] = e[i+1:]
		}
	}

	return environment
}

func getPortsEnvironmentVariables() []string {
	allEnv := getAllEnvironmentVariables()

	var portsEnvs []string
	for key := range allEnv {
		if strings.HasPrefix(key, ENV_PREFIX) {
			portsEnvs = append(portsEnvs, allEnv[key])
		}
	}

	return portsEnvs
}

func parseEnvPort(envValue string) (portForward *PortForward, err error) {
	chunks := strings.Split(envValue, ":")
	if len(chunks) < 2 {
		err = fmt.Errorf("bad port mapping \"%s\": should at least contain REMOTE_HOST:REMOTE_PORT", envValue)
		return
	}

	remotePort, err := strconv.ParseInt(chunks[len(chunks)-1], 10, 64)
	remoteHost := chunks[len(chunks)-2]
	if err != nil {
		err = fmt.Errorf("bad port mapping \"%s\": invalid Remote Port", envValue)
		return
	}

	var localPort int64
	if len(chunks) == 3 {
		localPort, err = strconv.ParseInt(chunks[len(chunks)-3], 10, 64)
		if err != nil {
			err = fmt.Errorf("bad port mapping \"%s\": invalid Local Port", envValue)
			return
		}
	} else {
		localPort = remotePort
	}

	portForward = &PortForward{
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		LocalPort:  localPort,
	}
	return
}

func LoadPorts() (ports []*PortForward, errors []error) {
	portsEnvVars := getPortsEnvironmentVariables()
	for _, value := range portsEnvVars {
		parsedPort, err := parseEnvPort(value)

		if err == nil {
			ports = append(ports, parsedPort)
		} else {
			errors = append(errors, err)
		}
	}

	return
}
