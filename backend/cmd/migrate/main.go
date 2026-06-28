package main

import (
	"log/slog"
	"os"

	"github.com/pms/backend/internal/config"
	"github.com/pms/backend/internal/db"
)

func main() {
	cfg := config.Load()
	if err := db.Migrate(cfg.MigrateDatabaseURL); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}
