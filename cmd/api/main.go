package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	// GET /stats/days resolves IANA zones; embed the database so the binary does
	// not depend on the host's zoneinfo.
	_ "time/tzdata"

	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/httpserver"
	"github.com/EmilM32/vynno-api/internal/mail"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	slog.SetDefault(newLogger(cfg.LogFormat))

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("database open", "err", err)
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		slog.Error("database ping", "err", err)
		os.Exit(1)
	}
	if err := store.Migrate(db); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}

	mailer, err := mail.New(cfg.Mail.Mode, mail.SMTP{
		Host:     cfg.Mail.Host,
		Port:     cfg.Mail.Port,
		Username: cfg.Mail.Username,
		Password: cfg.Mail.Password,
		StartTLS: cfg.Mail.StartTLS,
		From:     cfg.Mail.From,
		FromName: cfg.Mail.FromName,
	})
	if err != nil {
		slog.Error("mailer", "err", err)
		os.Exit(1)
	}

	svc := service.New(store.NewPostgres(db), mailer)
	if cfg.RateLimitMode == "relaxed" {
		svc.Limits = service.RelaxedAuthLimits()
		slog.Warn("auth rate limits relaxed", "database", "vynno_dev")
	}

	r := httpserver.NewRouter(svc, httpserver.Options{
		SPAOrigins:      cfg.SPAOrigins,
		CookieSecure:    cfg.CookieSecure,
		PublicAPIOrigin: cfg.PublicAPIOrigin,
		TrustedProxies:  cfg.TrustedProxies,
		Ready:           db.PingContext,
	})

	// ReadTimeout covers the largest body (a 2 MiB avatar multipart); WriteTimeout
	// the slowest handler (bcrypt plus SMTP). IdleTimeout closes parked keep-alives.
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	sweepCtx, stopSweep := context.WithCancel(context.Background())
	defer stopSweep()
	go sweepExpired(sweepCtx, svc)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listen", "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	case sig := <-sigCh:
		slog.Info("shutdown", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown", "err", err)
			os.Exit(1)
		}
	}
}

// sweepExpired removes expired session tokens and stale one-time codes at start
// and then every service.SweepInterval until ctx is cancelled.
func sweepExpired(ctx context.Context, svc *service.Service) {
	ticker := time.NewTicker(service.SweepInterval)
	defer ticker.Stop()
	for {
		res, err := svc.SweepExpired(ctx)
		switch {
		case err != nil && ctx.Err() == nil:
			slog.Error("sweep", "err", err)
		case err == nil && (res.Tokens > 0 || res.Challenges > 0):
			slog.Info("sweep", "tokens", res.Tokens, "challenges", res.Challenges)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func newLogger(format string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
