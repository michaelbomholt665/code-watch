package main

import (
	"os"

	"circular/internal/cliapp"
)

func main() {
	os.Exit(cliapp.Run(os.Args[1:]))
}
