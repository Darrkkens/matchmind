package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"matchmind/internal/ai"
	"matchmind/internal/api"
	"matchmind/internal/football"
	"matchmind/internal/service"
	"matchmind/internal/store"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	if path := loadDotEnv(); path != "" {
		slog.Info("loaded environment file", "path", path)
	}
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	timeout, err := time.ParseDuration(env("OLLAMA_TIMEOUT", "240s"))
	if err != nil || timeout <= 0 || timeout > 10*time.Minute {
		return errors.New("OLLAMA_TIMEOUT must be between 0 and 10m")
	}
	client, err := ai.NewOllama(env("OLLAMA_URL", "http://localhost:11434"), env("OLLAMA_MODEL", "gemma3:4b"), timeout)
	if err != nil {
		return err
	}
	openProvider, err := football.NewOpenFootballProvider(env("OPENFOOTBALL_LEAGUE", "br.1"), env("OPENFOOTBALL_SEASON", "2026"))
	if err != nil {
		return err
	}
	// Optional PostgreSQL cache for upstream API responses (see docker-compose.yml).
	var cache football.ResponseCache = football.NewMemoryCache()
	if url := os.Getenv("DATABASE_URL"); url != "" {
		pg, err := store.OpenPostgres(context.Background(), url)
		if err != nil {
			return fmt.Errorf("DATABASE_URL: %w", err)
		}
		defer pg.Close()
		cache = pg
		slog.Info("response cache", "backend", "postgres")
	}
	// AlmanacStats: free, keyless; personal/non-commercial use only (see README). Takes precedence.
	if env("ALMANACSTATS", "off") == "on" && env("OPENFOOTBALL_LEAGUE", "br.1") == "br.1" {
		almanac := football.NewAlmanacStats(cache)
		openProvider.SetStatistics(almanac)
		openProvider.SetSquad(almanac)
		slog.Info("match statistics enabled", "source", almanac.Name())
	} else if key := os.Getenv("API_FUTEBOL_KEY"); key != "" {
		stats, err := football.NewAPIFutebolStats(key, env("OPENFOOTBALL_LEAGUE", "br.1"), cache)
		if err != nil {
			return err
		}
		openProvider.SetStatistics(stats)
		slog.Info("match statistics enabled", "source", stats.Name())
	} else if key := os.Getenv("API_FOOTBALL_KEY"); key != "" {
		stats, err := football.NewAPIFootballStats(key, env("OPENFOOTBALL_LEAGUE", "br.1"), env("OPENFOOTBALL_SEASON", "2026"))
		if err != nil {
			return err
		}
		openProvider.SetStatistics(stats)
		slog.Info("match statistics enabled", "source", stats.Name())
	}
	// Série A history since 2003 (GPL-2.0 dataset, fetched at runtime). Disable with BRASILEIRAO_HISTORY=off.
	if env("OPENFOOTBALL_LEAGUE", "br.1") == "br.1" && env("BRASILEIRAO_HISTORY", "on") != "off" {
		openProvider.SetHistory(football.NewHistoryProvider())
	}
	// Optional local FBref CSV export (never downloaded or redistributed); see README.
	if dir := os.Getenv("FBREF_DIR"); dir != "" && env("OPENFOOTBALL_LEAGUE", "br.1") == "br.1" {
		if season, err := football.LoadFBref(dir); err != nil {
			slog.Warn("FBref season stats disabled", "error", err)
		} else {
			openProvider.SetSeason(season)
			slog.Info("season statistics enabled", "source", "fbref-local", "as_of", season.AsOf())
		}
	}
	teams := &service.TeamService{Provider: openProvider, Source: "openfootball", Notice: football.OpenFootballNotice, Analyst: client}
	chat := &service.ChatService{Teams: teams, AI: client}
	server := &http.Server{Addr: env("API_ADDR", "127.0.0.1:8080"), Handler: api.New(teams, chat, strings.Split(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ",")), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: timeout + 20*time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("MatchMind API ready", "address", server.Addr, "provider", teams.Source)
		errCh <- server.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
	}
	return nil
}
