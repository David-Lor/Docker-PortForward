package main

import (
	"fmt"
	"os/exec"
	"sync"
)

func forwardPort(port *PortForward) {
	fmt.Printf("Forwarding port %s ...\n", port.ToString())

	// socat TCP-LISTEN:80,fork TCP:202.54.1.5:80
	localChunk := fmt.Sprintf("tcp-listen:%d,reuseaddr,fork", port.LocalPort)
	remoteChunk := fmt.Sprintf("tcp:%s:%d", port.RemoteHost, port.RemotePort)

	cmd := exec.Command("socat", localChunk, remoteChunk)
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("Port forward for mapping %s failed with error: %s\n", port.ToString(), err.Error())
	} else {
		fmt.Printf("Port forward for mapping %s closed without error\n", port.ToString())
	}
}

func ForwardPorts(ports []*PortForward) {
	var waitGroup sync.WaitGroup

	for _, port := range ports {
		waitGroup.Add(1)

		go func(port *PortForward) {
			defer waitGroup.Done()
			forwardPort(port)
		}(port)
	}

	waitGroup.Wait()
}
