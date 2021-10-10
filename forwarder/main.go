package main

import (
	"fmt"
	"os"
)

func main() {
	ports, errors := LoadPorts()
	if errors != nil {
		fmt.Println("Invalid port mapping/s found:")
		for _, err := range errors {
			fmt.Println(err.Error())
		}

		os.Exit(1)
	}

	if len(ports) == 0 {
		fmt.Println("No ports configured!")
		os.Exit(1)
	}

	ForwardPorts(ports)
}
