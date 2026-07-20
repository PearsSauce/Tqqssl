package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PearsSauce/Tqqssl/backend/internal/config"
	"github.com/PearsSauce/Tqqssl/backend/internal/httpapi"
	"github.com/PearsSauce/Tqqssl/backend/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()
	st, err := store.Open(cfg.DataFile)
	if err != nil {
		logger.Error("open store failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	api := httpapi.New(cfg, st, logger)
	logger.Info("starting tqqssl personal api", "addr", cfg.Addr, "data_file", cfg.DataFile)
	if err := httpapi.ListenAndServe(ctx, cfg, api.Routes()); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
