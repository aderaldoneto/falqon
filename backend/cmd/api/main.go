package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/aderaldo/falqon/backend/internal/api"
)

func main() {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to configure database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	apiServer := api.NewServer(pool)
	strictHandler := api.NewStrictHandler(apiServer, nil)
	api.HandlerFromMux(strictHandler, router)

	address := os.Getenv("ADDRESS")
	if address == "" {
		address = "0.0.0.0:8080"
	}

	server := &http.Server{
		Addr:              address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownContext, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("API listening", "address", address)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownContext.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("failed to shut down API", "error", err)
	}
}
