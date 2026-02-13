package main

import (
	"os"

	"circular/internal/ui/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
