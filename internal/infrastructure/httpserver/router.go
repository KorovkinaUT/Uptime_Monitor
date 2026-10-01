package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(monitors monitorUseCase, checks checkUseCase, webDir string, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	monitorHandler := NewMonitorHandler(monitors, logger)
	checkHandler := NewCheckHandler(checks, logger)
	router := chi.NewRouter()
	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/monitors", monitorHandler.List)
		api.Post("/monitors", monitorHandler.Create)
		api.Get("/monitors/{id}", monitorHandler.Get)
		api.Put("/monitors/{id}", monitorHandler.Update)
		api.Delete("/monitors/{id}", monitorHandler.Delete)
		api.Get("/monitors/{id}/checks", checkHandler.History)
		api.Post("/monitors/{id}/check", checkHandler.Run)
	})

	staticHandler := http.FileServer(http.Dir(webDir))
	router.Handle("/*", staticHandler)
	return router
}

func requestPathParam(request *http.Request, name string) string {
	return chi.URLParam(request, name)
}
