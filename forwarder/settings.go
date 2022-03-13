package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env var format: PORT=localport:remotehost:remoteport
const (
	EnvPrefix     = "PORT"
	EnvSocksProxy = "SOCKS_PROXY"
)

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

func getPortsEnvironmentVariables(allEnv map[string]string) map[string]string {
	portsEnvs := make(map[string]string)
	for key, value := range allEnv {
		if strings.HasPrefix(key, EnvPrefix) {
			portsEnvs[key] = value
		}
	}

	return portsEnvs
}

func parsePortValue(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

func parsePortRangeValue(value string) (ok bool, start int64, end int64, count int64, err error) {
	chunks := strings.Split(value, "-")
	if len(chunks) != 2 {
		ok = false
		return
	}

	err = func() (inErr error) {
		start, inErr = parsePortValue(chunks[0])
		if inErr != nil {
			return
		}
		end, inErr = parsePortValue(chunks[1])
		return inErr
	}()

	if err == nil {
		count = end - start + 1
		if count <= 0 {
			err = fmt.Errorf("non-sequential range")
			return
		}
		ok = true
	}

	return
}

func tryParseEnvPortRange(localPortChunk string, remoteHostChunk string, remotePortChunk string) (ok bool, portsForwards []*PortForward, err error) {
	ok, startRemoteRange, _, countRemoteRange, err := parsePortRangeValue(remotePortChunk)
	if !ok && err == nil {
		return
	}
	if err != nil {
		err = fmt.Errorf("could not parse REMOTE port range: %s", err)
		return
	}

	var startLocalRange int64
	var okLocal bool
	if localPortChunk != "" {
		var countLocalRange int64
		okLocal, startLocalRange, _, countLocalRange, err = parsePortRangeValue(localPortChunk)
		if err != nil {
			err = fmt.Errorf("could not parse LOCAL port range: %s", err)
			return
		}

		if okLocal && countLocalRange != countRemoteRange {
			err = fmt.Errorf("the port ranges do not have the same length on local/remote (local=%d remote=%d)", countLocalRange, countRemoteRange)
			return
		}
	}

	if !okLocal {
		startLocalRange = startRemoteRange
	}

	var i int64
	for i = 0; i < countRemoteRange; i++ {
		localPort := startLocalRange + i
		remotePort := startRemoteRange + i

		portForward := &PortForward{
			LocalPort:  localPort,
			RemoteHost: remoteHostChunk,
			RemotePort: remotePort,
		}
		portsForwards = append(portsForwards, portForward)
	}
	return
}

func parseSimpleEnvPort(localPortChunk string, remoteHostChunk string, remotePortChunk string) (portForward *PortForward, err error) {
	remotePort, err := parsePortValue(remotePortChunk)
	if err != nil {
		err = fmt.Errorf("invalid REMOTE port: %s", err)
		return
	}

	localPort := remotePort
	if localPortChunk != "" {
		localPort, err = parsePortValue(localPortChunk)
		if err != nil {
			err = fmt.Errorf("invalid LOCAL port: %s", err)
			return
		}
	}

	portForward = &PortForward{
		LocalPort:  localPort,
		RemoteHost: remoteHostChunk,
		RemotePort: remotePort,
	}
	return
}

func parseEnvPort(envValue string) (portsForwards []*PortForward, err error) {
	chunks := strings.Split(envValue, ":")
	if len(chunks) < 2 {
		err = fmt.Errorf("should at least contain REMOTE_HOST:REMOTE_PORT")
		return
	}

	remotePortChunk := chunks[len(chunks)-1]
	remoteHostChunk := chunks[len(chunks)-2]
	localPortChunk := ""
	if len(chunks) > 2 {
		localPortChunk = chunks[len(chunks)-3]
	}

	// Port range
	isPortRange, portsForwards, err := tryParseEnvPortRange(localPortChunk, remoteHostChunk, remotePortChunk)
	if err != nil || isPortRange {
		return
	}

	// Simple port
	portForward, err := parseSimpleEnvPort(localPortChunk, remoteHostChunk, remotePortChunk)
	if err != nil {
		return
	}
	portsForwards = append(portsForwards, portForward)
	return
}

func loadPorts(allEnv map[string]string) (ports []*PortForward, errors []error) {
	portsEnvVars := getPortsEnvironmentVariables(allEnv)
	for key, value := range portsEnvVars {
		parsedPorts, err := parseEnvPort(value)

		if err == nil {
			ports = append(ports, parsedPorts...)
		} else {
			errors = append(errors, fmt.Errorf("invalid port mapping \"%s=%s\": %s", key, value, err))
		}
	}

	return
}

func loadSocksProxy(allEnv map[string]string) (proxy *SocksProxy, err error) {
	rawProxy := allEnv[EnvSocksProxy]
	if rawProxy == "" {
		return
	}

	chunks := strings.Split(rawProxy, ":")
	if len(chunks) != 2 {
		err = fmt.Errorf("invalid socks proxy, must be in format 'ip:port'")
		return
	}

	host := chunks[0]
	port, err := strconv.ParseInt(chunks[1], 10, 32)
	if err != nil {
		err = fmt.Errorf("invalid socks proxy port: %s", err)
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

	socksProxy, errSocksProxy := loadSocksProxy(allEnv)
	if errSocksProxy != nil {
		errors = append(errors, errSocksProxy)
	}

	if errors != nil {
		return
	}

	settings = &Settings{
		Ports:      ports,
		SocksProxy: socksProxy,
	}
	return
}
