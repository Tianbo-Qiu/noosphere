package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Tianbo-Qiu/noosphere/internal/db"
	"github.com/Tianbo-Qiu/noosphere/internal/logging"
	"github.com/Tianbo-Qiu/noosphere/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger, err := logging.New(os.Stderr, getenv("LOG_FORMAT", "text"), getenv("LOG_LEVEL", "info"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	if err := run(context.Background(), logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	dbURL := getenv("DATABASE_URL", "")
	if dbURL == "" {
		return errors.New("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	logger.Info("database connected")

	secureCookies, err := strconv.ParseBool(getenv("COOKIE_SECURE", "true"))
	if err != nil {
		return fmt.Errorf("COOKIE_SECURE: %w", err)
	}

	config := web.Config{
		SecureCookies: secureCookies,
		FlowLifetime:  time.Hour,
	}
	srv, err := web.NewServer(config, db.New(pool), logger)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	addr := getenv("ADDR", ":8080")
	logger.Info("listening",
		"addr", addr,
		"secure_cookies", config.SecureCookies,
		"flow_lifetime", config.FlowLifetime,
	)
	return http.ListenAndServe(addr, srv.Handler())
}
