package main

import (
	"fmt"
	"os/exec"
	"sync"
)

func getPortForwardArgs(port *PortForward) []string {
	// socat TCP-LISTEN:80,fork TCP:202.54.1.5:80
	localChunk := fmt.Sprintf("TCP-LISTEN:%d,fork", port.LocalPort)
	remoteChunk := fmt.Sprintf("TCP:%s:%d", port.RemoteHost, port.RemotePort)
	return []string{localChunk, remoteChunk}
}

func getPortForwardSocksProxyArgs(port *PortForward, proxy *SocksProxy) []string {
	// socat TCP-LISTEN:80,fork SOCKS4A:localhost:202.54.1.5:80,socksport=10000 (being proxy @ localhost:10000)
	localChunk := fmt.Sprintf("TCP-LISTEN:%d,fork", port.LocalPort)
	remoteChunk := fmt.Sprintf("SOCKS4A:%s:%s:%d,socksport=%d", proxy.Host, port.RemoteHost, port.RemotePort, proxy.Port)
	return []string{localChunk, remoteChunk}
}

func forwardPort(port *PortForward, socksProxy *SocksProxy) {
	fmt.Printf("Forwarding port %s ...\n", port.ToString())

	var cmdArgs []string
	if socksProxy == nil {
		cmdArgs = getPortForwardArgs(port)
	} else {
		cmdArgs = getPortForwardSocksProxyArgs(port, socksProxy)
	}

	cmd := exec.Command("socat", cmdArgs...)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Port forward for mapping %s failed with error: %s\n", port.ToString(), err.Error())
	} else {
		fmt.Printf("Port forward for mapping %s closed without error\n", port.ToString())
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
