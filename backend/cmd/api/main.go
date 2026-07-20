package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/PearsSauce/Tqqssl/backend/internal/acmeaccount"
	"github.com/PearsSauce/Tqqssl/backend/internal/config"
	"github.com/PearsSauce/Tqqssl/backend/internal/httpapi"
	"github.com/PearsSauce/Tqqssl/backend/internal/secretbox"
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
	box, err := secretbox.Open(cfg.SecretKeyFile)
	if err != nil {
		logger.Error("open secret key file failed", "error", err)
		os.Exit(1)
	}
	if migrated, err := st.EncryptPlaintextDNSSecrets(box.Encrypt, secretbox.IsCiphertext); err != nil {
		logger.Error("migrate dns account secrets failed", "error", err)
		os.Exit(1)
	} else if migrated > 0 {
		logger.Info("migrated plaintext dns account secrets", "count", migrated)
	}
	accountKey, err := acmeaccount.Open(cfg.ACMEAccountKeyFile)
	if err != nil {
		logger.Error("open acme account key failed", "error", err)
		os.Exit(1)
	}
	logger.Info("acme account key ready", "key_type", accountKey.Type(), "key_file", cfg.ACMEAccountKeyFile)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	api := httpapi.New(cfg, st, box, logger)
	logger.Info("starting tqqssl personal api", "addr", cfg.Addr, "data_file", cfg.DataFile)
	if err := httpapi.ListenAndServe(ctx, cfg, api.Routes()); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
