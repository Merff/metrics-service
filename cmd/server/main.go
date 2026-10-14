package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"metrics-service/internal/config"
	"metrics-service/internal/repository"
	"metrics-service/internal/service"
	"metrics-service/internal/storage"
	transport "metrics-service/internal/transport/http"
)

const shutdownTimeout = 10 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	if err := storage.MigrateUp(cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	repo := repository.NewMetricRepository(pool)
	svc := service.NewMetricService(repo)
	handler := transport.NewMetricHandler(svc)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           transport.NewRouter(handler, pool),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
