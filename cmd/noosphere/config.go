package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/web"
)

// config holds the server's settings.
type config struct {
	DatabaseURL string
	Addr        string
	LogFormat   string
	LogLevel    string
	Web         web.Config
}

// loadConfig reads the settings from environment variables, fills in the
// defaults and rejects values that cannot be used.
func loadConfig() (config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return config{}, errors.New("DATABASE_URL is not set")
	}
	secureCookies, err := strconv.ParseBool(getenv("COOKIE_SECURE", "true"))
	if err != nil {
		return config{}, fmt.Errorf("COOKIE_SECURE: %w", err)
	}
	return config{
		DatabaseURL: databaseURL,
		Addr:        getenv("ADDR", ":8080"),
		LogFormat:   getenv("LOG_FORMAT", "text"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		Web: web.Config{
			SecureCookies: secureCookies,
			FlowLifetime:  time.Hour,
		},
	}, nil
}

// getenv returns the environment variable key, or fallback if it is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
