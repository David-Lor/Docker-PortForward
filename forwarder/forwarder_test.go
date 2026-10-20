package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetPortForwardArgs(t *testing.T) {
	port := &PortForward{LocalPort: 8080, RemoteHost: "10.0.0.5", RemotePort: 80}
	assert.Equal(t, []string{"TCP-LISTEN:8080,fork,reuseaddr", "TCP:10.0.0.5:80"}, getPortForwardArgs(port))
}

func TestGetPortForwardSocksProxyArgs(t *testing.T) {
	port := &PortForward{LocalPort: 8080, RemoteHost: "example.com", RemotePort: 80}
	proxy := &SocksProxy{Host: "localhost", Port: 9050}
	assert.Equal(t,
		[]string{"TCP-LISTEN:8080,fork,reuseaddr", "SOCKS4A:localhost:example.com:80,socksport=9050"},
		getPortForwardSocksProxyArgs(port, proxy),
	)
}

func TestNextBackoff(t *testing.T) {
	// First restart
	assert.Equal(t, restartBackoffMin, nextBackoff(0, 0))
	// Quick failures double the delay, up to the maximum
	assert.Equal(t, 2*time.Second, nextBackoff(1*time.Second, 100*time.Millisecond))
	assert.Equal(t, 4*time.Second, nextBackoff(2*time.Second, 0))
	assert.Equal(t, 8*time.Second, nextBackoff(4*time.Second, 0))
	assert.Equal(t, 10*time.Second, nextBackoff(8*time.Second, 0))
	assert.Equal(t, 10*time.Second, restartBackoffMax)
	// Once capped, it keeps retrying at the maximum
	assert.Equal(t, restartBackoffMax, nextBackoff(restartBackoffMax, 0))
	assert.Equal(t, restartBackoffMax, nextBackoff(nextBackoff(restartBackoffMax, 0), 0))
	// A long healthy run resets the delay
	assert.Equal(t, restartBackoffMin, nextBackoff(restartBackoffMax, restartBackoffResetAfter))
}
