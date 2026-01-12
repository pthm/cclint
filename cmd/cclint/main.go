package main

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/pthm/cclint/internal/cmd"
)

func main() {
	if err := fang.Execute(context.Background(), cmd.RootCmd); err != nil {
		os.Exit(1)
	}
}
