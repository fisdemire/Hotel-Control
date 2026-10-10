package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/fisdemire/Hotel-Control/config"
	"github.com/fisdemire/Hotel-Control/internal/app"
	"github.com/fisdemire/Hotel-Control/internal/platform/postgres"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	return app.New(cfg, db).Run(ctx)
}
