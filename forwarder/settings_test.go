package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSettings(t *testing.T) {
	t.Run("s1", func(t *testing.T) {
		env := map[string]string{
			"PORT0": "9990:10.10.10.0:9090",
			"PORT1": "9991:10.10.10.1:9091",
			"PORT2": "10.10.10.2:9092",
			"port2": "localhost:9999", // ignored
			"bla":   "localhost:9998", // ignored
		}
		expectedSettings := &Settings{
			Ports: []*PortForward{
				{
					LocalPort:  9990,
					RemoteHost: "10.10.10.0",
					RemotePort: 9090,
				},
				{
					LocalPort:  9991,
					RemoteHost: "10.10.10.1",
					RemotePort: 9091,
				},
				{
					LocalPort:  9092,
					RemoteHost: "10.10.10.2",
					RemotePort: 9092,
				},
			},
		}
		runnerTestLoadSettings(t, env, expectedSettings, nil)
	})

	t.Run("s2", func(t *testing.T) {
		env := map[string]string{
			"PORT0": "9000:host1:9000",
			"PORT1": "9001",
			"PORT2": "host1:wololo",
			"PORT3": "foo:host1:9000",
			"PORT4": "8000:127.0.0.1",
		}
		expectedErrors := []string{
			"invalid port mapping \"PORT1=9001\": should at least contain REMOTE_HOST:REMOTE_PORT",
			"invalid port mapping \"PORT2=host1:wololo\": invalid REMOTE port: strconv.ParseInt: parsing \"wololo\": invalid syntax",
			"invalid port mapping \"PORT3=foo:host1:9000\": invalid LOCAL port: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"invalid port mapping \"PORT4=8000:127.0.0.1\": invalid REMOTE port: strconv.ParseInt: parsing \"127.0.0.1\": invalid syntax",
		}
		runnerTestLoadSettings(t, env, nil, expectedErrors)
	})

	t.Run("s3", func(t *testing.T) {
		env := map[string]string{
			"PORT":        "host1:9000",
			"SOCKS_PROXY": "tor:9050",
		}
		expectedSettings := &Settings{
			Ports: []*PortForward{
				{
					LocalPort:  9000,
					RemoteHost: "host1",
					RemotePort: 9000,
				},
			},
			SocksProxy: &SocksProxy{
				Host: "tor",
				Port: 9050,
			},
		}
		runnerTestLoadSettings(t, env, expectedSettings, nil)
	})

	t.Run("s4", func(t *testing.T) {
		env := map[string]string{
			"PORTS":       "nginx:80",
			"SOCKS_PROXY": "9050",
		}
		expectedErrors := []string{
			"invalid socks proxy, must be in format 'ip:port'",
		}
		runnerTestLoadSettings(t, env, nil, expectedErrors)
	})

	t.Run("s5", func(t *testing.T) {
		env := map[string]string{
			"SOCKS_PROXY": "nginx:foo",
		}
		expectedErrors := []string{
			"invalid socks proxy port: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"no ports defined",
		}
		runnerTestLoadSettings(t, env, nil, expectedErrors)
	})

	t.Run("s6", func(t *testing.T) {
		env := map[string]string{
			"PORTRNG1":    "8000-8002:host1:9015-9017",
			"PORTRNGinv1": "localhost:7000-foo", // invalid (bad format)
			"PORTRNG2":    "host2:7000-7002",
		}
		expectedSettings := &Settings{
			Ports: []*PortForward{
				// host1
				{
					LocalPort:  8000,
					RemoteHost: "host1",
					RemotePort: 9015,
				},
				{
					LocalPort:  8001,
					RemoteHost: "host1",
					RemotePort: 9016,
				},
				{
					LocalPort:  8002,
					RemoteHost: "host1",
					RemotePort: 9017,
				},
				// host2
				{
					LocalPort:  7000,
					RemoteHost: "host2",
					RemotePort: 7000,
				},
				{
					LocalPort:  7001,
					RemoteHost: "host2",
					RemotePort: 7001,
				},
				{
					LocalPort:  7002,
					RemoteHost: "host2",
					RemotePort: 7002,
				},
			},
		}
		expectedErrors := []string{
			"invalid port mapping \"PORTRNGinv1=localhost:7000-foo\": could not parse REMOTE port range: strconv.ParseInt: parsing \"foo\": invalid syntax",
		}
		runnerTestLoadSettings(t, env, expectedSettings, expectedErrors)
	})

	t.Run("s7", func(t *testing.T) {
		env := map[string]string{
			"PORTRNG1": "localhost:7000-foo",
			"PORTRNG2": "7000-foo:localhost:7000-8000",
			"PORTRNG3": "7000-8000:localhost:7000-foo",
			"PORTRNG4": "7000-foo:localhost:7000-foo",
			"PORTRNG5": "7000-8000:localhost:7000-7500",
			"PORTRNG6": "7000-7500:localhost:7000-8000",
			"PORTRNG7": "7000-8000:localhost",
		}
		expectedErrors := []string{
			"invalid port mapping \"PORTRNG1=localhost:7000-foo\": could not parse REMOTE port range: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"invalid port mapping \"PORTRNG2=7000-foo:localhost:7000-8000\": could not parse LOCAL port range: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"invalid port mapping \"PORTRNG3=7000-8000:localhost:7000-foo\": could not parse REMOTE port range: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"invalid port mapping \"PORTRNG4=7000-foo:localhost:7000-foo\": could not parse REMOTE port range: strconv.ParseInt: parsing \"foo\": invalid syntax",
			"invalid port mapping \"PORTRNG5=7000-8000:localhost:7000-7500\": the port ranges do not have the same length on local/remote (local=1001 remote=501)",
			"invalid port mapping \"PORTRNG6=7000-7500:localhost:7000-8000\": the port ranges do not have the same length on local/remote (local=501 remote=1001)",
			"invalid port mapping \"PORTRNG7=7000-8000:localhost\": invalid REMOTE port: strconv.ParseInt: parsing \"localhost\": invalid syntax",
		}
		runnerTestLoadSettings(t, env, nil, expectedErrors)
	})
}

func settingstestSetup(env map[string]string) {
	for k, v := range env {
		err := os.Setenv(k, v)
		if err != nil {
			panic(err)
		}
	}
}

func settingstestTeardown(env map[string]string) {
	for k := range env {
		err := os.Unsetenv(k)
		if err != nil {
			panic(err)
		}
	}
}

func runnerTestLoadSettings(t *testing.T, env map[string]string, expectedSettings *Settings, expectedErrors []string) {
	settingstestSetup(env)
	defer settingstestTeardown(env)

	resultSettings, resultErrs := LoadSettings()

	if len(expectedErrors) > 0 {
		var resultErrsStrs []string
		for _, err := range resultErrs {
			resultErrsStrs = append(resultErrsStrs, err.Error())
		}

		assert.ElementsMatch(t, expectedErrors, resultErrsStrs)
		return
	}

	assert.Empty(t, resultErrs)
	assert.Equal(t, expectedSettings.SocksProxy, resultSettings.SocksProxy)
	assert.ElementsMatch(t, expectedSettings.Ports, resultSettings.Ports)
}
