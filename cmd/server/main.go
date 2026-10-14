package main

import (
	"log"
	"net/http"
	_ "time/tzdata" // the scratch image has no zoneinfo; needed to resolve iCal TZIDs

	"calabunga/internal/api"
	"calabunga/internal/config"
	"calabunga/web"
)

func main() {
	cfg := config.Load()
	router := api.NewRouter(cfg, web.Handler())
	log.Printf("listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
