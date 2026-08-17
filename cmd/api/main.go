package main

import (
	"log"

	"github.com/EmilM32/vynno-api/internal/config"
	"github.com/EmilM32/vynno-api/internal/httpserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	r := httpserver.NewRouter()
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
