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
	router.Get("/monitors", h.monitorHandler.List)
	router.Post("/monitors", h.monitorHandler.Create)
	router.Get("/monitors/{id}", h.monitorHandler.Get)
	router.Put("/monitors/{id}", h.monitorHandler.Update)
	router.Delete("/monitors/{id}", h.monitorHandler.Delete)
	router.Get("/monitors/{id}/checks", h.checkHandler.History)
	router.Post("/monitors/{id}/check", h.checkHandler.Run)

	router.Handle("/*", http.FileServer(http.Dir(h.webDir)))
}

// requestPathParam returns a named route parameter
func requestPathParam(request *http.Request, name string) string {
	return chi.URLParam(request, name)
}
