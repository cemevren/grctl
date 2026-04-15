package main

import (
	"os"

	"grctl/server/cmd/grctld/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
