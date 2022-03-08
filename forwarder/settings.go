package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env var format: PORT=localport:remotehost:remoteport
const ENV_PREFIX = "PORT"
const ENV_SOCKS_PROXY = "SOCKS_PROXY"

type PortForward struct {
	LocalPort  int64
	RemoteHost string
	RemotePort int64
}

type SocksProxy struct {
	Host string
	Port int
}

type Settings struct {
	Ports      []*PortForward
	SocksProxy *SocksProxy
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

func getPortsEnvironmentVariables(allEnv map[string]string) []string {
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

func loadPorts(allEnv map[string]string) (ports []*PortForward, errors []error) {
	portsEnvVars := getPortsEnvironmentVariables(allEnv)
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

func loadSocksProxy(allEnv map[string]string) (proxy *SocksProxy, errors []error) {
	rawProxy := allEnv[ENV_SOCKS_PROXY]
	if rawProxy == "" {
		return
	}

	chunks := strings.Split(rawProxy, ":")
	if len(chunks) != 2 {
		errors = []error{fmt.Errorf("unvalid socks proxy, must be in format 'ip:port'")}
		return
	}

	host := chunks[0]
	port, err := strconv.ParseInt(chunks[1], 10, 32)
	if err != nil {
		errors = []error{fmt.Errorf("unvalid socks proxy port: %s", err)}
		return
	}

	proxy = &SocksProxy{
		Host: host,
		Port: int(port),
	}
	return
}

func LoadSettings() (settings *Settings, errors []error) {
	allEnv := getAllEnvironmentVariables()

	ports, errors := loadPorts(allEnv)
	if len(ports) == 0 && len(errors) == 0 {
		errors = append(errors, fmt.Errorf("no ports defined"))
	}

	socksProxy, errors2 := loadSocksProxy(allEnv)
	errors = append(errors, errors2...)

	if errors != nil {
		return
	}

	settings = &Settings{
		Ports:      ports,
		SocksProxy: socksProxy,
	}
	return
}
