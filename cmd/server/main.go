package main

import (
	"log"
	"net/http"

	"calabunga/internal/api"
	"calabunga/internal/config"
)

func main() {
	cfg := config.Load()
	router := api.NewRouter(cfg)
	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
