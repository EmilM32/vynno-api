package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/devdata"
	"github.com/EmilM32/vynno-api/internal/domain"
	"github.com/EmilM32/vynno-api/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "devdata: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 1 || (args[0] != "reset" && args[0] != "seed") {
		return fmt.Errorf("usage: devdata reset|seed")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	username, err := domain.NormalizeUsername(cfg.BootstrapUsername)
	if err != nil {
		return fmt.Errorf("bootstrap username: %w", err)
	}
	password, err := domain.NormalizePassword(cfg.BootstrapPassword)
	if err != nil {
		return fmt.Errorf("bootstrap password: %w", err)
	}
	seedPass := strings.TrimSpace(os.Getenv("SEED_PASSWORD"))
	if seedPass == "" {
		seedPass = devdata.DefaultSeedPassword
	}
	if _, err := domain.NormalizePassword(seedPass); err != nil {
		return fmt.Errorf("SEED_PASSWORD: %w", err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	if err := store.Migrate(db); err != nil {
		return err
	}

	opts := devdata.Options{
		BootstrapUsername: username,
		BootstrapPassword: password,
		SeedPassword:      seedPass,
	}
	var ds devdata.Dataset
	switch args[0] {
	case "reset":
		ds = devdata.BuildReset(opts)
	case "seed":
		ds = devdata.BuildSeed(opts)
	}

	ctx := context.Background()
	if err := devdata.Wipe(ctx, db); err != nil {
		return err
	}
	if err := devdata.Apply(ctx, store.NewPostgres(db), ds); err != nil {
		return err
	}

	fmt.Printf("%s\n", args[0])
	for _, acc := range ds.Accounts {
		fmt.Printf("  %-10s  %s  %s  (%d projects, %d sessions)\n",
			acc.Username, acc.Password, acc.Blurb, len(acc.Projects), len(acc.Sessions))
	}
	return nil
}
