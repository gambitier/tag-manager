package main

import (
	"os"

	"github.com/gambitier/tag-manager/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
