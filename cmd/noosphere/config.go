package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/pwned"
	"github.com/Tianbo-Qiu/noosphere/internal/web"
)

// config holds the server's settings.
type config struct {
	DatabaseURL string
	Addr        string
	LogFormat   string
	LogLevel    string
	Web         web.Config

	// PasswordContextCheck and PwnedPasswordsCheck turn on the password
	// checks against context words and breached passwords. It is on by
	// default; local development turns it off, so sign-up works offline and
	// sends no requests to an outside service.
	PwnedPasswordsCheck   bool
	PwnedPasswordsURL     string
	PwnedPasswordsTimeout time.Duration
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
	contextCheck, err := strconv.ParseBool(getenv("PASSWORD_CONTEXT_CHECK", "true"))
	if err != nil {
		return config{}, fmt.Errorf("PASSWORD_CONTEXT_CHECK: %w", err)
	}
	strengthMeter, err := strconv.ParseBool(getenv("PASSWORD_STRENGTH_METER", "false"))
	if err != nil {
		return config{}, fmt.Errorf("PASSWORD_STRENGTH_METER: %w", err)
	}
	pwnedCheck, err := strconv.ParseBool(getenv("PWNED_PASSWORDS_CHECK", "true"))
	if err != nil {
		return config{}, fmt.Errorf("PWNED_PASSWORDS_CHECK: %w", err)
	}
	pwnedTimeout, err := time.ParseDuration(getenv("PWNED_PASSWORDS_TIMEOUT", "2s"))
	if err != nil {
		return config{}, fmt.Errorf("PWNED_PASSWORDS_TIMEOUT: %w", err)
	}
	return config{
		DatabaseURL: databaseURL,
		Addr:        getenv("ADDR", ":8080"),
		LogFormat:   getenv("LOG_FORMAT", "text"),
		LogLevel:    getenv("LOG_LEVEL", "info"),
		Web: web.Config{
			SecureCookies:         secureCookies,
			FlowLifetime:          time.Hour,
			PasswordContextCheck:  contextCheck,
			PasswordStrengthMeter: strengthMeter,
		},
		PwnedPasswordsCheck:   pwnedCheck,
		PwnedPasswordsURL:     getenv("PWNED_PASSWORDS_URL", pwned.DefaultURL),
		PwnedPasswordsTimeout: pwnedTimeout,
	}, nil
}

// getenv returns the environment variable key, or fallback if it is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
