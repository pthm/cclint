package main

import (
	"os"

	"github.com/pthm-cable/cclint/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
