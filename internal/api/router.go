package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"calabunga/internal/config"
)

func NewRouter(cfg config.Config, webHandler http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/connect", NewConnectHandler(cfg).ServeHTTP)

	r.Group(func(r chi.Router) {
		r.Use(JWTMiddleware(cfg))
		r.Get("/api/calendars", NewCalendarsHandler(cfg).ServeHTTP)
		r.Get("/api/events", NewEventsHandler(cfg).ServeHTTP)
		r.Post("/api/events", NewEventsHandler(cfg).ServeHTTP)
		r.Put("/api/events/{uid}", NewEventsHandler(cfg).ServeHTTP)
		r.Delete("/api/events/{uid}", NewEventsHandler(cfg).ServeHTTP)
	})

	r.Handle("/*", webHandler)
	return r
}
