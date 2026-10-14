package main

import (
	"context"
	"log"
	"time"

	"metrics-service/internal/config"
	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/storage"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := storage.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	defer pool.Close()

	repo := repository.NewMetricRepository(pool)

	err = repo.Insert(
		ctx,
		models.Metric{
			ClientID:         1,
			AppID:            1,
			Requests:         200,
			Landings:         50,
			ApiRegisterTotal: 100,
			ApiRegisterScam:  10,
			Timestamp:        time.Now(),
		},
	)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("metric inserted")
}
