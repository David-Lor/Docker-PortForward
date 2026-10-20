package main

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	restartBackoffMin = 1 * time.Second
	restartBackoffMax = 10 * time.Second
	// A run lasting at least this long is considered healthy, resetting the backoff
	restartBackoffResetAfter = 1 * time.Minute
)

func getPortForwardArgs(port *PortForward) []string {
	// socat TCP-LISTEN:80,fork,reuseaddr TCP:202.54.1.5:80
	localChunk := fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", port.LocalPort)
	remoteChunk := fmt.Sprintf("TCP:%s:%d", port.RemoteHost, port.RemotePort)
	return []string{localChunk, remoteChunk}
}

func getPortForwardSocksProxyArgs(port *PortForward, proxy *SocksProxy) []string {
	// socat TCP-LISTEN:80,fork,reuseaddr SOCKS4A:localhost:202.54.1.5:80,socksport=10000 (being proxy @ localhost:10000)
	localChunk := fmt.Sprintf("TCP-LISTEN:%d,fork,reuseaddr", port.LocalPort)
	remoteChunk := fmt.Sprintf("SOCKS4A:%s:%s:%d,socksport=%d", proxy.Host, port.RemoteHost, port.RemotePort, proxy.Port)
	return []string{localChunk, remoteChunk}
}

// nextBackoff returns the delay before restarting a socat process, given the previous delay and how long the last run lasted.
// socat is always restarted; the delay only grows up to restartBackoffMax.
func nextBackoff(previous time.Duration, lastRun time.Duration) time.Duration {
	if previous == 0 || lastRun >= restartBackoffResetAfter {
		return restartBackoffMin
	}
	next := previous * 2
	if next > restartBackoffMax {
		next = restartBackoffMax
	}
	return next
}

func forwardPort(port *PortForward, socksProxy *SocksProxy) {
	var cmdArgs []string
	if socksProxy == nil {
		cmdArgs = getPortForwardArgs(port)
	} else {
		cmdArgs = getPortForwardSocksProxyArgs(port, socksProxy)
	}

	var backoff time.Duration
	for {
		fmt.Printf("Forwarding port %s ...\n", port.ToString())

		// socat output goes straight to the container output, so its errors are visible.
		// No pipes are used, so the exit is noticed even if forked children are still alive.
		cmd := exec.Command("socat", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		startedAt := time.Now()
		err := cmd.Run()
		backoff = nextBackoff(backoff, time.Since(startedAt))

		if err != nil {
			fmt.Printf("Port forward for mapping %s failed with error: %s; restarting in %s\n", port.ToString(), err.Error(), backoff)
		} else {
			fmt.Printf("Port forward for mapping %s closed without error; restarting in %s\n", port.ToString(), backoff)
		}
		time.Sleep(backoff)
	}
}

func ForwardPorts(settings *Settings) {
	var waitGroup sync.WaitGroup

	for _, port := range settings.Ports {
		waitGroup.Add(1)

		go func(port *PortForward) {
			defer waitGroup.Done()
			forwardPort(port, settings.SocksProxy)
		}(port)
	}

	waitGroup.Wait()
}
