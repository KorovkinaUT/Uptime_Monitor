package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handler groups HTTP handlers and static web configuration
type handler struct {
	monitorHandler *MonitorHandler
	checkHandler   *CheckHandler
	webDir         string
}

// newHandler builds handlers from application dependencies
func newHandler(dependencies Dependencies) *handler {
	logger := dependencies.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &handler{
		monitorHandler: NewMonitorHandler(dependencies.Monitors, logger),
		checkHandler:   NewCheckHandler(dependencies.Checks, logger),
		webDir:         dependencies.WebDir,
	}
}

// registerRoutes binds API and frontend routes
func (h *handler) registerRoutes(router chi.Router) {
	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/monitors", h.monitorHandler.List)
		api.Post("/monitors", h.monitorHandler.Create)
		api.Get("/monitors/{id}", h.monitorHandler.Get)
		api.Put("/monitors/{id}", h.monitorHandler.Update)
		api.Delete("/monitors/{id}", h.monitorHandler.Delete)
		api.Get("/monitors/{id}/checks", h.checkHandler.History)
		api.Post("/monitors/{id}/check", h.checkHandler.Run)
	})

	router.Handle("/*", http.FileServer(http.Dir(h.webDir)))
}

// requestPathParam returns a named route parameter
func requestPathParam(request *http.Request, name string) string {
	return chi.URLParam(request, name)
}
