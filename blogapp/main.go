package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := loadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, err := openStore(ctx, cfg)
	if err != nil {
		slog.Error("could not open the archive", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := os.MkdirAll(cfg.MediaDir, 0o755); err != nil {
		slog.Warn("could not create media directory", "dir", cfg.MediaDir, "error", err)
	}

	render := NewRenderer(cfg)
	app := newApp(cfg, store, render)

	if cfg.Seed {
		if err := seedIfEmpty(ctx, app); err != nil {
			slog.Warn("seed content failed", "error", err)
		}
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           app,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("ARNΛV is on the press", "addr", cfg.Addr, "site", cfg.SiteURL, "dev", cfg.Dev)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "error", err)
	}
	slog.Info("the presses are quiet")
}
