package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Tianbo-Qiu/noosphere/internal/logging"
	"github.com/Tianbo-Qiu/noosphere/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, err := logging.New(os.Stderr, cfg.LogFormat, cfg.LogLevel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	if err := run(context.Background(), cfg, logger); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config, logger *slog.Logger) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("create pool: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	logger.Info("database connected")

	srv, err := web.NewServer(cfg.Web, pool, logger)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	logger.Info("listening",
		"addr", cfg.Addr,
		"secure_cookies", cfg.Web.SecureCookies,
		"flow_lifetime", cfg.Web.FlowLifetime,
	)
	return http.ListenAndServe(cfg.Addr, srv.Handler())
}
