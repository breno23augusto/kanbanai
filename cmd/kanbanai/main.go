package main

import (
	"kanbanai/internal/adapter/in/cli"
	"log/slog"
	"os"
)

func main() {
	if err := cli.Execute(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}
