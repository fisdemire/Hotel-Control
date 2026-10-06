package main

import (
	"context"
	"log"

	"github.com/fisdemire/Hotel-Control/config"
	"github.com/fisdemire/Hotel-Control/internal/app"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := pgxpool.New(context.Background(), cfg.Postgres.DSN())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}

	a := app.New(cfg, db)

	if err := a.Run(); err != nil {
		log.Fatal(err)
	}
}
