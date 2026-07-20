package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr           string
	DataFile       string
	FrontendOrigin string
	SessionTTL     time.Duration
}

func Load() Config {
	return Config{
		Addr:           envString("TQQSSL_ADDR", ":8080"),
		DataFile:       envString("TQQSSL_DATA_FILE", "data/tqqssl-personal.json"),
		FrontendOrigin: strings.TrimRight(envString("TQQSSL_FRONTEND_ORIGIN", "http://localhost:5173"), "/"),
		SessionTTL:     time.Duration(envInt("TQQSSL_SESSION_TTL_HOURS", 24)) * time.Hour,
	}
}

func envString(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
