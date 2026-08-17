package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/httpserver"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	if err := store.Migrate(db); err != nil {
		log.Fatal(err)
	}

	pg := store.NewPostgres(db)
	ctx := context.Background()
	if err := pg.SeedEmpty(ctx, store.DefaultUserID(), store.DefaultProfile(), store.DefaultProject()); err != nil {
		log.Fatal(err)
	}
	userID, ok, err := pg.FirstUserID(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if !ok {
		log.Fatal("no user after seed")
	}

	r := httpserver.NewRouter(service.New(pg, userID))
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
