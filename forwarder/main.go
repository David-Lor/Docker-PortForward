package main

import (
	"fmt"
	"os"
)

func main() {
	settings, errors := LoadSettings()
	if errors != nil {
		fmt.Println("Errors in settings:")
		for _, err := range errors {
			fmt.Println(err.Error())
		}

		os.Exit(1)
	}

	ForwardPorts(settings)
}
