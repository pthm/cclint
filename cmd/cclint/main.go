package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/pthm/cclint/internal/cmd"
)

func main() {
	err := fang.Execute(context.Background(), cmd.RootCmd)

	// Show update notice after command execution (works even when commands fail)
	cmd.ShowUpdateNoticeIfAvailable()

	if err != nil {
		os.Exit(1)
	}
}
