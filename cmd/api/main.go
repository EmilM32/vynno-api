package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/httpserver"
	"github.com/EmilM32/vynno-api/internal/service"
	"github.com/EmilM32/vynno-api/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
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

	username, err := domain.NormalizeUsername(cfg.BootstrapUsername)
	if err != nil {
		log.Fatal(err)
	}
	password, err := domain.NormalizePassword(cfg.BootstrapPassword)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	pg := store.NewPostgres(db)
	ctx := context.Background()
	if err := pg.Bootstrap(ctx, store.DefaultUserID(), username, string(hash), store.DefaultProfile(), store.DefaultProject()); err != nil {
		log.Fatal(err)
	}

	r := httpserver.NewRouter(service.New(pg), httpserver.Options{
		SPAOrigins:      cfg.SPAOrigins,
		CookieSecure:    cfg.CookieSecure,
		PublicAPIOrigin: cfg.PublicAPIOrigin,
	})
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
